// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

const pipelineCIMobileAutomationStepUse = "mobile-automation"

var pipelineCIRunnerHealthCheck = checkPipelineCIRunnerHealth

type tempRecordDeleteInput struct {
	ExpectedOwnerID    string `json:"expected_owner_id"`
	ExpectedIdentifier string `json:"expected_identifier"`
}

type pipelineCITempRecordResult struct {
	Record     *core.Record
	Identifier string
}

type pipelineCITempRecordsOptions struct {
	Refs           []string
	Collection     string
	IdentifierKey  string
	IdentifierName string
	ResourceDomain string
	ResourceName   string
	OwnerID        string
	CommitSHA      string
	HostURL        string
	RelationField  string
	UniqueName     func(core.App, string, string, string) string
}

type pipelineCIBaseRequest struct {
	PipelineIdentifier string
	CommitSHA          string
	Metadata           map[string]any
	IDs                []string
	DeviceID           string
	DeviceType         string
	HostURL            string
}

type pipelineCIRunContext struct {
	OrganizationRecord *core.Record
	UserID             string
	UserName           string
	UserEmail          string
	PipelineRecord     *core.Record
	PipelineYAML       string
}

// handleTempRecordDelete deletes a temporary CI record after route-specific validation.
func handleTempRecordDelete(collection, resourceName string) func(*core.RequestEvent) error {
	resourceDomain := strings.ReplaceAll(resourceName, " ", "_")
	return func(e *core.RequestEvent) error {
		recordID := strings.TrimSpace(e.Request.PathValue("record"))
		if recordID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"record",
				resourceName+" record id is required",
				"missing record path parameter",
			)
		}

		var input tempRecordDeleteInput
		if e.Request.Body != nil {
			if err := json.NewDecoder(e.Request.Body).Decode(&input); err != nil &&
				!errors.Is(err, io.EOF) {
				return apierror.New(
					http.StatusBadRequest,
					resourceDomain,
					"invalid delete validation payload",
					err.Error(),
				)
			}
		}

		record, err := e.App.FindRecordById(collection, recordID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return e.JSON(http.StatusOK, map[string]any{"deleted": false})
			}
			return apierror.New(
				http.StatusInternalServerError,
				resourceDomain,
				"failed to find "+resourceName,
				err.Error(),
			)
		}

		if apiErr := validateTempRecordDeleteRequest(
			e.App,
			record,
			input,
			resourceName,
			resourceDomain,
		); apiErr != nil {
			return apiErr
		}

		if err := e.App.Delete(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				resourceDomain,
				"failed to delete "+resourceName,
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, map[string]any{"deleted": true})
	}
}

func validateTempRecordDeleteRequest(
	app core.App,
	record *core.Record,
	input tempRecordDeleteInput,
	resourceName string,
	resourceDomain string,
) *apierror.APIError {
	expectedOwnerID := strings.TrimSpace(input.ExpectedOwnerID)
	expectedIdentifier := strings.TrimSpace(input.ExpectedIdentifier)
	if expectedOwnerID == "" || expectedIdentifier == "" {
		return apierror.New(
			http.StatusBadRequest,
			resourceDomain,
			"delete validation payload is required",
			"expected_owner_id and expected_identifier are required",
		)
	}
	if record.GetString("owner") != expectedOwnerID {
		return apierror.New(
			http.StatusForbidden,
			resourceDomain,
			"temporary "+resourceName+" owner mismatch",
			resourceName+" owner does not match expected_owner_id",
		)
	}
	resolved, err := canonify.Resolve(app, expectedIdentifier)
	if err != nil {
		return apierror.New(
			http.StatusForbidden,
			resourceDomain,
			"temporary "+resourceName+" identifier mismatch",
			err.Error(),
		)
	}
	if resolved.Id != record.Id {
		return apierror.New(
			http.StatusForbidden,
			resourceDomain,
			"temporary "+resourceName+" identifier mismatch",
			"expected_identifier does not resolve to the requested record",
		)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizePipelineCIIdentifiers(values []string) []string {
	seen := map[string]struct{}{}
	var identifiers []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			identifier := canonify.NormalizePath(part)
			if identifier == "" {
				continue
			}
			if _, ok := seen[identifier]; ok {
				continue
			}
			seen[identifier] = struct{}{}
			identifiers = append(identifiers, identifier)
		}
	}
	return identifiers
}

func metadataSHA(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	sha, _ := metadata["sha"].(string)
	return strings.TrimSpace(sha)
}

func ensureMetadataSHA(metadata map[string]any, sha string) map[string]any {
	if metadata == nil {
		metadata = map[string]any{}
	}
	if strings.TrimSpace(metadataSHA(metadata)) == "" && strings.TrimSpace(sha) != "" {
		metadata["sha"] = strings.TrimSpace(sha)
	}
	return metadata
}

func metadataFromForm(req *http.Request) (map[string]any, *apierror.APIError) {
	raw := strings.TrimSpace(req.FormValue("metadata"))
	if raw == "" {
		return nil, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"metadata",
			"metadata must be valid JSON",
			err.Error(),
		)
	}
	return metadata, nil
}

func parsePipelineCIBaseRequest(
	e *core.RequestEvent,
	idsKey string,
	hostURLKey string,
) (pipelineCIBaseRequest, *apierror.APIError) {
	contentType := e.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(e.Request.Body).Decode(&raw); err != nil {
			return pipelineCIBaseRequest{}, apierror.New(
				http.StatusBadRequest,
				"request",
				"invalid JSON input",
				err.Error(),
			)
		}
		metadata, apiErr := metadataFromJSONRaw(raw["metadata"])
		if apiErr != nil {
			return pipelineCIBaseRequest{}, apiErr
		}
		commitSHA := firstNonEmpty(stringFromJSONRaw(raw["commit_sha"]), metadataSHA(metadata))
		return pipelineCIBaseRequest{
			PipelineIdentifier: strings.TrimSpace(stringFromJSONRaw(raw["pipeline_identifier"])),
			CommitSHA:          strings.TrimSpace(commitSHA),
			Metadata:           ensureMetadataSHA(metadata, commitSHA),
			IDs:                normalizePipelineCIIdentifiers(stringsFromJSONRaw(raw[idsKey])),
			DeviceID:           strings.TrimSpace(stringFromJSONRaw(raw["device_id"])),
			DeviceType:         strings.TrimSpace(stringFromJSONRaw(raw["device_type"])),
			HostURL:            strings.TrimSpace(stringFromJSONRaw(raw[hostURLKey])),
		}, nil
	}

	if err := e.Request.ParseForm(); err != nil {
		return pipelineCIBaseRequest{}, apierror.New(
			http.StatusBadRequest,
			"request",
			"failed to parse form request",
			err.Error(),
		)
	}
	metadata, apiErr := metadataFromForm(e.Request)
	if apiErr != nil {
		return pipelineCIBaseRequest{}, apiErr
	}
	commitSHA := firstNonEmpty(e.Request.FormValue("commit_sha"), metadataSHA(metadata))
	return pipelineCIBaseRequest{
		PipelineIdentifier: strings.TrimSpace(e.Request.FormValue("pipeline_identifier")),
		CommitSHA:          strings.TrimSpace(commitSHA),
		Metadata:           ensureMetadataSHA(metadata, commitSHA),
		IDs:                normalizePipelineCIIdentifiers(e.Request.Form[idsKey]),
		DeviceID:           strings.TrimSpace(e.Request.FormValue("device_id")),
		DeviceType:         strings.TrimSpace(e.Request.FormValue("device_type")),
		HostURL:            strings.TrimSpace(e.Request.FormValue(hostURLKey)),
	}, nil
}

func metadataFromJSONRaw(raw json.RawMessage) (map[string]any, *apierror.APIError) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"metadata",
			"metadata must be valid JSON",
			err.Error(),
		)
	}
	return metadata, nil
}

func stringFromJSONRaw(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}

func stringsFromJSONRaw(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return values
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return []string{value}
	}
	return nil
}

func validatePipelineCIBaseRequest(
	input pipelineCIBaseRequest,
	hostURLKey string,
) *apierror.APIError {
	if strings.TrimSpace(input.PipelineIdentifier) == "" {
		return apierror.New(
			http.StatusBadRequest,
			"pipeline_identifier",
			"pipeline_identifier is required",
			"missing pipeline_identifier",
		)
	}
	if strings.TrimSpace(input.CommitSHA) == "" {
		return apierror.New(
			http.StatusBadRequest,
			"metadata",
			"commit_sha or metadata.sha is required",
			"missing commit sha",
		)
	}
	parsedURL, err := url.Parse(strings.TrimSpace(input.HostURL))
	if err != nil || parsedURL == nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return apierror.New(
			http.StatusBadRequest,
			hostURLKey,
			hostURLKey+" is invalid",
			hostURLKey+" must be an http or https URL",
		)
	}
	if deviceType := strings.TrimSpace(input.DeviceType); deviceType != "" {
		if _, ok := walletAPKAllowedDeviceTypes[deviceType]; !ok {
			return apierror.New(
				http.StatusBadRequest,
				"device_type",
				"device_type is invalid",
				"device_type must be one of android_emulator, redroid, android_phone, ios_simulator, ios_phone",
			)
		}
	}
	return nil
}

func resolvePipelineCIRunContext(
	e *core.RequestEvent,
	pipelineIdentifier string,
) (pipelineCIRunContext, *apierror.APIError) {
	if e.Auth == nil {
		return pipelineCIRunContext{}, apierror.New(
			http.StatusUnauthorized,
			"auth",
			"authentication required",
			"user not authenticated",
		)
	}

	orgRecord, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
	if err != nil {
		return pipelineCIRunContext{}, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization record",
			err.Error(),
		)
	}
	pipelineRecord, err := canonify.Resolve(e.App, pipelineIdentifier)
	if err != nil {
		return pipelineCIRunContext{}, apierror.New(
			http.StatusNotFound,
			"pipeline_identifier",
			"pipeline not found",
			err.Error(),
		)
	}
	if apiErr := authorizePipelineRunWalletAPKPipeline(
		pipelineRecord,
		orgRecord.Id,
	); apiErr != nil {
		return pipelineCIRunContext{}, apiErr
	}
	pipelineYAML := strings.TrimSpace(pipelineRecord.GetString("yaml"))
	if pipelineYAML == "" {
		return pipelineCIRunContext{}, apierror.New(
			http.StatusBadRequest,
			"pipeline_yaml",
			"pipeline yaml is required",
			"pipeline has no yaml",
		)
	}

	return pipelineCIRunContext{
		OrganizationRecord: orgRecord,
		UserID:             e.Auth.Id,
		UserName:           e.Auth.GetString("name"),
		UserEmail:          e.Auth.GetString("email"),
		PipelineRecord:     pipelineRecord,
		PipelineYAML:       pipelineYAML,
	}, nil
}

func resolvePipelineCIDeviceID(
	ctx context.Context,
	app core.App,
	ownerID string,
	workflowDefinition *pipelineinternal.WorkflowDefinition,
	input pipelineCIBaseRequest,
) (string, bool, bool, *apierror.APIError) {
	hasStepRunner, needsGlobalRunner := pipelineCIMobileRunnerSelectionState(workflowDefinition)
	if !hasStepRunner && !needsGlobalRunner {
		return "", hasStepRunner, needsGlobalRunner, nil
	}
	if strings.TrimSpace(input.DeviceID) != "" {
		return input.DeviceID, hasStepRunner, needsGlobalRunner, nil
	}
	if strings.TrimSpace(input.DeviceType) == "" {
		if apiErr := validatePipelineCIGlobalDeviceRequest(
			"",
			hasStepRunner,
			needsGlobalRunner,
		); apiErr != nil {
			return "", hasStepRunner, needsGlobalRunner, apiErr
		}
		return "", hasStepRunner, needsGlobalRunner, nil
	}
	deviceID, apiErr := selectPipelineCIDeviceByType(ctx, app, ownerID, input.DeviceType)
	return deviceID, hasStepRunner, needsGlobalRunner, apiErr
}

func parsePipelineCIWorkflow(
	pipelineYAML string,
) (*pipelineinternal.WorkflowDefinition, *apierror.APIError) {
	workflowDefinition, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}

	return workflowDefinition, nil
}

func createTempPipelineCIRecord(
	e *core.RequestEvent,
	opts pipelineCITempRecordsOptions,
	original *core.Record,
	rewrittenYAML string,
) (*core.Record, string, *apierror.APIError) {
	collection := original.Collection()
	record := core.NewRecord(collection)
	fileFieldValues := make(map[string]interface{})
	for _, field := range collection.Fields {
		fieldName := field.GetName()
		if field.Type() == "file" && original.Get(fieldName) != nil {
			fileFieldValues[fieldName] = original.Get(fieldName)
		}
	}
	for key, value := range original.FieldsData() {
		if slices.Contains(systemFields, key) ||
			key == "canonified_name" ||
			key == "yaml" {
			continue
		}
		if _, ok := fileFieldValues[key]; ok {
			continue
		}
		record.Set(key, value)
	}

	tag := canonify.CanonifyPlain(opts.CommitSHA)
	if tag == "" {
		return nil, "", apierror.New(
			http.StatusBadRequest,
			"metadata",
			"commit sha is invalid",
			"commit sha cannot be canonified",
		)
	}
	record.Set("owner", opts.OwnerID)
	baseName := strings.TrimSpace(original.GetString("name"))
	if baseName == "" {
		baseName = opts.ResourceName
	}
	name := baseName + "-" + tag
	if opts.UniqueName != nil {
		name = opts.UniqueName(
			e.App,
			original.GetString("name"),
			tag,
			original.GetString(opts.RelationField),
		)
	}
	record.Set("name", name)
	record.Set("published", false)
	record.Set("yaml", rewrittenYAML)

	if err := e.App.Save(record); err != nil {
		return nil, "", apierror.New(
			http.StatusInternalServerError,
			opts.ResourceDomain,
			"failed to create temporary "+opts.ResourceName,
			err.Error(),
		)
	}
	if len(fileFieldValues) > 0 {
		if err := cloneFiles(e.App, original, record, fileFieldValues); err != nil {
			e.App.Logger().Warn(fmt.Sprintf(
				"failed to clone files for temporary %s %s: %v",
				opts.ResourceName,
				record.Id,
				err,
			))
		}
	}

	identifier, err := canonify.BuildPath(
		e.App,
		record,
		canonify.CanonifyPaths[collection.Name],
		"",
	)
	if err != nil {
		_ = e.App.Delete(record)
		return nil, "", apierror.New(
			http.StatusInternalServerError,
			opts.ResourceDomain,
			"failed to build temporary "+opts.ResourceName+" identifier",
			err.Error(),
		)
	}

	return record, identifier, nil
}

func rewriteStepCIHost(stepCIYAML string, hostURL string) (string, bool) {
	if strings.TrimSpace(stepCIYAML) == "" {
		return "", false
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(stepCIYAML), &doc); err != nil {
		return "", false
	}
	env, ok := doc["env"].(map[string]any)
	if !ok {
		return "", false
	}
	if _, ok := env["host"].(string); !ok {
		return "", false
	}
	env["host"] = hostURL
	rewritten, err := yaml.Marshal(doc)
	if err != nil {
		return "", false
	}
	return string(rewritten), true
}

func createPipelineCITempRecords(
	e *core.RequestEvent,
	opts pipelineCITempRecordsOptions,
) ([]pipelineCITempRecordResult, map[string]string, *apierror.APIError) {
	rewriteMap := map[string]string{}
	tempRecords := []pipelineCITempRecordResult{}
	for _, ref := range opts.Refs {
		record, apiErr := resolveCanonicalCollectionRecord(
			e.App,
			opts.Collection,
			ref,
			opts.IdentifierKey,
			opts.IdentifierName,
		)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempRecords, opts.IdentifierName)
			return nil, nil, apiErr
		}
		if apiErr := authorizeOwnedOrPublishedRecord(
			record,
			opts.OwnerID,
			opts.Collection,
			opts.ResourceDomain,
		); apiErr != nil {
			rollbackPipelineCITempRecords(e, tempRecords, opts.IdentifierName)
			return nil, nil, apiErr
		}

		rewrittenYAML, ok := rewriteStepCIHost(record.GetString("yaml"), opts.HostURL)
		if !ok {
			continue
		}
		tempRecord, identifier, apiErr := createTempPipelineCIRecord(
			e,
			opts,
			record,
			rewrittenYAML,
		)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempRecords, opts.IdentifierName)
			return nil, nil, apiErr
		}
		result := pipelineCITempRecordResult{Record: tempRecord, Identifier: identifier}
		tempRecords = append(tempRecords, result)
		rewriteMap[canonify.NormalizePath(ref)] = identifier
	}

	return tempRecords, rewriteMap, nil
}

func resolvePipelineCIReferenceFilter(
	app core.App,
	collection string,
	requestedIDs []string,
	field string,
	resourceName string,
) (map[string]struct{}, *apierror.APIError) {
	if len(requestedIDs) == 0 {
		return nil, nil
	}
	filter := map[string]struct{}{}
	for _, requestedID := range requestedIDs {
		if _, apiErr := resolveCanonicalCollectionRecord(
			app,
			collection,
			requestedID,
			field,
			resourceName,
		); apiErr != nil {
			return nil, apiErr
		}
		filter[canonify.NormalizePath(requestedID)] = struct{}{}
	}
	return filter, nil
}

func collectPipelineCIReferences(
	workflowDefinition *pipelineinternal.WorkflowDefinition,
	stepUse string,
	payloadKey string,
) []string {
	if workflowDefinition == nil {
		return nil
	}
	var refs []string
	collect := func(step pipelineinternal.StepSpec) {
		if step.Use != stepUse || step.With.Payload == nil {
			return
		}
		identifier, ok := step.With.Payload[payloadKey].(string)
		if !ok || strings.TrimSpace(identifier) == "" {
			return
		}
		refs = append(refs, canonify.NormalizePath(identifier))
	}
	for _, step := range workflowDefinition.Steps {
		collect(step.StepSpec)
		for _, onErr := range step.OnError {
			if onErr != nil {
				collect(onErr.StepSpec)
			}
		}
		for _, onSuccess := range step.OnSuccess {
			if onSuccess != nil {
				collect(onSuccess.StepSpec)
			}
		}
	}
	return refs
}

func rollbackPipelineCITempRecords(
	e *core.RequestEvent,
	tempRecords []pipelineCITempRecordResult,
	resourceName string,
) {
	for _, tempRecord := range tempRecords {
		if tempRecord.Record == nil || tempRecord.Record.Id == "" {
			continue
		}
		if err := e.App.Delete(tempRecord.Record); err != nil {
			e.App.Logger().Warn(fmt.Sprintf(
				"failed to rollback temporary %s %s: %v",
				resourceName,
				tempRecord.Record.Id,
				err,
			))
		}
	}
}

func rewritePipelineCIStepRefsYAML(
	pipelineYAML string,
	rewriteMap map[string]string,
	stepUse string,
	payloadKey string,
) (string, *apierror.APIError) {
	if len(rewriteMap) == 0 {
		return pipelineYAML, nil
	}
	workflowDefinition, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return "", apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}
	for i := range workflowDefinition.Steps {
		rewritePipelineCIStepRef(
			&workflowDefinition.Steps[i].StepSpec,
			rewriteMap,
			stepUse,
			payloadKey,
		)
		for _, onErr := range workflowDefinition.Steps[i].OnError {
			if onErr != nil {
				rewritePipelineCIStepRef(&onErr.StepSpec, rewriteMap, stepUse, payloadKey)
			}
		}
		for _, onSuccess := range workflowDefinition.Steps[i].OnSuccess {
			if onSuccess != nil {
				rewritePipelineCIStepRef(&onSuccess.StepSpec, rewriteMap, stepUse, payloadKey)
			}
		}
	}
	rewritten, err := yaml.Marshal(workflowDefinition)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"yaml",
			"failed to marshal pipeline yaml",
			err.Error(),
		)
	}
	return string(rewritten), nil
}

func rewritePipelineCIStepRef(
	step *pipelineinternal.StepSpec,
	rewriteMap map[string]string,
	stepUse string,
	payloadKey string,
) {
	if step == nil || step.Use != stepUse || step.With.Payload == nil {
		return
	}
	ref, ok := step.With.Payload[payloadKey].(string)
	if !ok {
		return
	}
	tempIdentifier, ok := rewriteMap[canonify.NormalizePath(ref)]
	if !ok {
		return
	}
	step.With.Payload[payloadKey] = tempIdentifier
}

func injectPipelineCIGlobalDeviceID(
	pipelineYAML string,
	workflowDefinition *pipelineinternal.WorkflowDefinition,
	deviceID string,
	hasStepRunner bool,
	needsGlobalRunner bool,
) (string, *apierror.APIError) {
	deviceID = canonify.NormalizePath(deviceID)
	if deviceID == "" {
		return pipelineYAML, nil
	}

	if hasStepRunner {
		return "", apierror.New(
			http.StatusBadRequest,
			"device_id",
			"device_id cannot be combined with step device_id",
			"remove step device_id values or omit device_id",
		)
	}
	if !needsGlobalRunner {
		return pipelineYAML, nil
	}

	workflowDefinition.Runtime.GlobalDeviceID = deviceID
	rewrittenYAML, err := yaml.Marshal(workflowDefinition)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"yaml",
			"failed to marshal pipeline yaml",
			err.Error(),
		)
	}

	return string(rewrittenYAML), nil
}

func validatePipelineCIGlobalDeviceRequest(
	deviceID string,
	hasStepRunner bool,
	needsGlobalRunner bool,
) *apierror.APIError {
	if !needsGlobalRunner || strings.TrimSpace(deviceID) != "" {
		return nil
	}
	if hasStepRunner {
		return apierror.New(
			http.StatusBadRequest,
			"device_id",
			"mobile-automation device_id configuration is incomplete",
			"pipeline mixes mobile-automation steps with device_id and steps without device_id; set device_id on every mobile-automation step or remove step device_id values and pass device_id or device_type",
		)
	}
	return apierror.New(
		http.StatusBadRequest,
		"device_id",
		"device_id or device_type is required",
		"pipeline has mobile-automation steps without device_id; pass device_id or device_type",
	)
}

func pipelineCIIgnoredDeviceWarning(
	deviceID string,
	deviceType string,
	hasStepRunner bool,
	needsGlobalRunner bool,
) string {
	if hasStepRunner || needsGlobalRunner {
		return ""
	}
	if strings.TrimSpace(deviceID) == "" && strings.TrimSpace(deviceType) == "" {
		return ""
	}
	return "device_id and device_type are ignored because pipeline has no mobile-automation steps"
}

func pipelineCIMobileRunnerSelectionState(
	workflowDefinition *pipelineinternal.WorkflowDefinition,
) (bool, bool) {
	if workflowDefinition == nil {
		return false, false
	}

	hasStepRunner := false
	needsGlobalRunner := false
	check := func(step pipelineinternal.StepSpec) {
		if step.Use != pipelineCIMobileAutomationStepUse {
			return
		}
		deviceID, _ := step.With.Payload["device_id"].(string)
		if strings.TrimSpace(deviceID) == "" {
			needsGlobalRunner = true
			return
		}
		hasStepRunner = true
	}

	for _, step := range workflowDefinition.Steps {
		check(step.StepSpec)
		for _, onErr := range step.OnError {
			if onErr != nil {
				check(onErr.StepSpec)
			}
		}
		for _, onSuccess := range step.OnSuccess {
			if onSuccess != nil {
				check(onSuccess.StepSpec)
			}
		}
	}

	return hasStepRunner, needsGlobalRunner
}

func selectPipelineCIDeviceByType(
	ctx context.Context,
	app core.App,
	ownerID string,
	deviceType string,
) (string, *apierror.APIError) {
	filter := "type = {:type}"
	params := dbx.Params{"type": deviceType}
	devices, err := app.FindRecordsByFilter(
		"mobile_devices",
		filter,
		"",
		-1,
		0,
		params,
	)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"device_type",
			"failed to list mobile devices",
			err.Error(),
		)
	}

	selectedDeviceID := ""
	selectedBacklog := 0
	for _, device := range devices {
		runner, runnerErr := app.FindRecordById("mobile_runners", device.GetString("runner"))
		if runnerErr != nil || (runner.GetString("owner") != ownerID && !runner.GetBool("published")) {
			continue
		}
		online, apiErr := pipelineCIRunnerOnline(ctx, runner)
		if apiErr != nil {
			return "", apiErr
		}
		if !online {
			continue
		}
		deviceID, deviceErr := mobileDeviceIdentifier(app, device)
		if deviceErr != nil {
			return "", apierror.New(http.StatusInternalServerError, "device_id", "failed_to_build_device_id", deviceErr.Error())
		}
		backlog, apiErr := pipelineCIDeviceBacklog(ctx, deviceID)
		if apiErr != nil {
			return "", apiErr
		}
		if selectedDeviceID == "" || backlog < selectedBacklog || (backlog == selectedBacklog && deviceID < selectedDeviceID) {
			selectedDeviceID = deviceID
			selectedBacklog = backlog
		}
	}
	if selectedDeviceID == "" {
		return "", apierror.New(
			http.StatusServiceUnavailable,
			"device_type",
			"no online device found for device_type",
			"no online mobile device matches "+deviceType,
		)
	}
	return selectedDeviceID, nil
}

func partitionPipelineCIRunnerCandidates(
	records []*core.Record,
	ownerID string,
) ([]*core.Record, []*core.Record) {
	ownedPrivateRecords := make([]*core.Record, 0, len(records))
	publishedRecords := make([]*core.Record, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(ownerID) != "" &&
			record.GetString("owner") == ownerID &&
			!record.GetBool("published") {
			ownedPrivateRecords = append(ownedPrivateRecords, record)
			continue
		}
		if record.GetBool("published") {
			publishedRecords = append(publishedRecords, record)
		}
	}

	return ownedPrivateRecords, publishedRecords
}

func selectOnlinePipelineCIRunner(
	ctx context.Context,
	app core.App,
	records []*core.Record,
) (string, *apierror.APIError) {
	selectedDeviceID := ""
	selectedBacklog := 0
	for _, record := range records {
		online, apiErr := pipelineCIRunnerOnline(ctx, record)
		if apiErr != nil {
			return "", apiErr
		}
		if !online {
			continue
		}

		deviceID, apiErr := pipelineCIDeviceID(record, app)
		if apiErr != nil {
			return "", apiErr
		}
		backlog, apiErr := pipelineCIDeviceBacklog(ctx, deviceID)
		if apiErr != nil {
			return "", apiErr
		}
		if selectedDeviceID == "" ||
			backlog < selectedBacklog ||
			(backlog == selectedBacklog && deviceID < selectedDeviceID) {
			selectedDeviceID = deviceID
			selectedBacklog = backlog
		}
	}

	return selectedDeviceID, nil
}

func pipelineCIDeviceID(record *core.Record, app core.App) (string, *apierror.APIError) {
	deviceID, err := mobileDeviceIdentifier(app, record)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"device_id",
			"failed to build device_id",
			err.Error(),
		)
	}
	if deviceID == "" {
		return "", apierror.New(
			http.StatusInternalServerError,
			"device_id",
			"failed to build device_id",
			"empty device_id",
		)
	}

	return deviceID, nil
}

func pipelineCIRunnerOnline(ctx context.Context, record *core.Record) (bool, *apierror.APIError) {
	runnerURL := mobileRunnerURL(record)
	if runnerURL == "" {
		return false, nil
	}

	online, err := pipelineCIRunnerHealthCheck(ctx, runnerURL)
	if err != nil {
		return false, apierror.New(
			http.StatusInternalServerError,
			"device_type",
			"failed to check runner health",
			err.Error(),
		)
	}

	return online, nil
}

func checkPipelineCIRunnerHealth(ctx context.Context, runnerURL string) (bool, error) {
	healthURL, err := url.JoinPath(runnerURL, "health")
	if err != nil {
		return false, err
	}

	healthCtx, cancel := context.WithTimeout(ctx, walletAPKRunnerHealthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(healthCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func pipelineCIDeviceBacklog(ctx context.Context, deviceID string) (int, *apierror.APIError) {
	state, err := queryMobileDeviceSemaphoreState(ctx, deviceID)
	if err != nil {
		if errors.Is(err, errSemaphoreNotFound) {
			return 0, nil
		}
		return 0, apierror.New(
			http.StatusInternalServerError,
			"device_type",
			"failed to query runner queue",
			err.Error(),
		)
	}

	return state.QueueLen + state.SlotsUsed, nil
}
