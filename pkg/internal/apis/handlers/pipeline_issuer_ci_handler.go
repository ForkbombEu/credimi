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
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

const issuerCICredentialOfferStepUse = "credential-offer"
const issuerCITempCredentialsConfigKey = "temp_credentials"

type pipelineRunIssuerRequest struct {
	PipelineIdentifier string
	CommitSHA          string
	Metadata           map[string]any
	IssuerID           string
	RunnerID           string
	RunnerType         string
	IssuerURL          string
}

type PipelineRunIssuerResponse struct {
	PipelineQueueResponse
	TempCredentials    []TempCredentialResponse `json:"temp_credentials,omitempty"`
	PipelineIdentifier string                   `json:"pipeline_identifier,omitempty"`
}

type TempCredentialResponse struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
}

type pipelineRunIssuerContext struct {
	input              pipelineRunIssuerRequest
	organizationRecord *core.Record
	userID             string
	userName           string
	userEmail          string
	pipelineRecord     *core.Record
	pipelineYAML       string
	credentialRefs     []issuerCredentialReference
}

type issuerCredentialReference struct {
	StepID       string
	CredentialID string
}

type tempCredential struct {
	Record     *core.Record
	Identifier string
}

func HandlePipelineRunIssuer() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, apiErr := parsePipelineRunIssuerRequest(e)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		if apiErr := validatePipelineRunIssuerRequest(input); apiErr != nil {
			return apiErr.JSON(e)
		}

		runContext, apiErr := resolvePipelineRunIssuerContext(e, input)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		tempCredentials, rewriteMap, apiErr := createPipelineRunIssuerTempCredentials(e, runContext)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		rewrittenYAML, apiErr := rewritePipelineRunIssuerYAML(runContext.pipelineYAML, rewriteMap)
		if apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return apiErr.JSON(e)
		}
		workflowDefinition, apiErr := parsePipelineCIWorkflow(rewrittenYAML)
		if apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return apiErr.JSON(e)
		}
		runnerID, hasStepRunner, needsGlobalRunner, apiErr := resolvePipelineRunIssuerRunnerID(
			e.Request.Context(),
			e.App,
			workflowDefinition,
			input,
		)
		if apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return apiErr.JSON(e)
		}
		manipulatedYAML, apiErr := injectPipelineCIGlobalRunnerID(
			rewrittenYAML,
			workflowDefinition,
			runnerID,
			hasStepRunner,
			needsGlobalRunner,
		)
		if apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return apiErr.JSON(e)
		}

		queueResponse, apiErr := enqueuePipelineRun(e, pipelineQueueRunContext{
			pipelineRecord:     runContext.pipelineRecord,
			pipelineIdentifier: input.PipelineIdentifier,
			organizationRecord: runContext.organizationRecord,
			userID:             runContext.userID,
			userName:           runContext.userName,
			userEmail:          runContext.userEmail,
			yaml:               manipulatedYAML,
			metadata:           input.Metadata,
			runType:            pipelineinternal.RunTypeCI,
			cleanup:            buildPipelineRunIssuerCleanupMetadata(tempCredentials),
		})
		if apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return apiErr.JSON(e)
		}

		return e.JSON(http.StatusOK, buildPipelineRunIssuerResponse(
			queueResponse,
			tempCredentials,
			input.PipelineIdentifier,
		))
	}
}

func buildPipelineRunIssuerResponse(
	queueResponse PipelineQueueResponse,
	tempCredentials []tempCredential,
	pipelineIdentifier string,
) PipelineRunIssuerResponse {
	if queueResponse.Position != nil {
		position := *queueResponse.Position + 1
		queueResponse.Position = &position
	}
	response := PipelineRunIssuerResponse{
		PipelineQueueResponse: queueResponse,
		PipelineIdentifier:    pipelineIdentifier,
	}
	for _, credential := range tempCredentials {
		if credential.Record == nil {
			continue
		}
		response.TempCredentials = append(response.TempCredentials, TempCredentialResponse{
			ID:         credential.Record.Id,
			Identifier: credential.Identifier,
		})
	}
	return response
}

func parsePipelineRunIssuerRequest(e *core.RequestEvent) (pipelineRunIssuerRequest, *apierror.APIError) {
	contentType := e.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var input struct {
			PipelineIdentifier string         `json:"pipeline_identifier"`
			CommitSHA          string         `json:"commit_sha"`
			CommitSHAAlt       string         `json:"commitSha"`
			Metadata           map[string]any `json:"metadata"`
			IssuerID           string         `json:"issuer_id"`
			RunnerID           string         `json:"runner_id"`
			RunnerType         string         `json:"runner_type"`
			IssuerURL          string         `json:"issuer_url"`
			TempIssuerURL      string         `json:"temp_issuer_url"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&input); err != nil {
			return pipelineRunIssuerRequest{}, apierror.New(
				http.StatusBadRequest,
				"request",
				"invalid JSON input",
				err.Error(),
			)
		}
		commitSHA := firstNonEmpty(input.CommitSHA, input.CommitSHAAlt, metadataSHA(input.Metadata))
		return pipelineRunIssuerRequest{
			PipelineIdentifier: strings.TrimSpace(input.PipelineIdentifier),
			CommitSHA:          strings.TrimSpace(commitSHA),
			Metadata:           ensureMetadataSHA(input.Metadata, commitSHA),
			IssuerID:           strings.TrimSpace(input.IssuerID),
			RunnerID:           strings.TrimSpace(input.RunnerID),
			RunnerType:         strings.TrimSpace(input.RunnerType),
			IssuerURL:          strings.TrimSpace(firstNonEmpty(input.IssuerURL, input.TempIssuerURL)),
		}, nil
	}

	if err := e.Request.ParseForm(); err != nil {
		return pipelineRunIssuerRequest{}, apierror.New(
			http.StatusBadRequest,
			"request",
			"failed to parse form request",
			err.Error(),
		)
	}
	metadata, apiErr := metadataFromForm(e.Request)
	if apiErr != nil {
		return pipelineRunIssuerRequest{}, apiErr
	}
	commitSHA := firstNonEmpty(
		e.Request.FormValue("commit_sha"),
		e.Request.FormValue("commitSha"),
		metadataSHA(metadata),
	)
	return pipelineRunIssuerRequest{
		PipelineIdentifier: strings.TrimSpace(e.Request.FormValue("pipeline_identifier")),
		CommitSHA:          strings.TrimSpace(commitSHA),
		Metadata:           ensureMetadataSHA(metadata, commitSHA),
		IssuerID:           strings.TrimSpace(e.Request.FormValue("issuer_id")),
		RunnerID:           strings.TrimSpace(e.Request.FormValue("runner_id")),
		RunnerType:         strings.TrimSpace(e.Request.FormValue("runner_type")),
		IssuerURL: strings.TrimSpace(firstNonEmpty(
			e.Request.FormValue("issuer_url"),
			e.Request.FormValue("temp_issuer_url"),
		)),
	}, nil
}

func validatePipelineRunIssuerRequest(input pipelineRunIssuerRequest) *apierror.APIError {
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
	issuerURL := strings.TrimSpace(input.IssuerURL)
	parsedURL, err := url.Parse(issuerURL)
	if err != nil || parsedURL == nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return apierror.New(
			http.StatusBadRequest,
			"issuer_url",
			"issuer_url is invalid",
			"issuer_url must be an http or https URL",
		)
	}
	if runnerType := strings.TrimSpace(input.RunnerType); runnerType != "" {
		if _, ok := walletAPKAllowedRunnerTypes[runnerType]; !ok {
			return apierror.New(
				http.StatusBadRequest,
				"runner_type",
				"runner_type is invalid",
				"runner_type must be one of android_emulator, redroid, android_phone, ios_simulator, ios_phone",
			)
		}
	}
	return nil
}

func resolvePipelineRunIssuerContext(
	e *core.RequestEvent,
	input pipelineRunIssuerRequest,
) (pipelineRunIssuerContext, *apierror.APIError) {
	if e.Auth == nil {
		return pipelineRunIssuerContext{}, apierror.New(
			http.StatusUnauthorized,
			"auth",
			"authentication required",
			"user not authenticated",
		)
	}

	orgRecord, err := GetUserOrganization(e.App, e.Auth.Id)
	if err != nil {
		return pipelineRunIssuerContext{}, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization record",
			err.Error(),
		)
	}
	pipelineRecord, err := canonify.Resolve(e.App, input.PipelineIdentifier)
	if err != nil {
		return pipelineRunIssuerContext{}, apierror.New(
			http.StatusNotFound,
			"pipeline_identifier",
			"pipeline not found",
			err.Error(),
		)
	}
	if apiErr := authorizePipelineRunWalletAPKPipeline(pipelineRecord, orgRecord.Id); apiErr != nil {
		return pipelineRunIssuerContext{}, apiErr
	}
	pipelineYAML := strings.TrimSpace(pipelineRecord.GetString("yaml"))
	if pipelineYAML == "" {
		return pipelineRunIssuerContext{}, apierror.New(
			http.StatusBadRequest,
			"pipeline_yaml",
			"pipeline yaml is required",
			"pipeline has no yaml",
		)
	}

	refs, apiErr := resolvePipelineRunIssuerCredentialReferences(e.App, pipelineYAML, input.IssuerID)
	if apiErr != nil {
		return pipelineRunIssuerContext{}, apiErr
	}

	return pipelineRunIssuerContext{
		input:              input,
		organizationRecord: orgRecord,
		userID:             e.Auth.Id,
		userName:           e.Auth.GetString("name"),
		userEmail:          e.Auth.GetString("email"),
		pipelineRecord:     pipelineRecord,
		pipelineYAML:       pipelineYAML,
		credentialRefs:     refs,
	}, nil
}

func resolvePipelineRunIssuerCredentialReferences(
	app core.App,
	pipelineYAML string,
	issuerID string,
) ([]issuerCredentialReference, *apierror.APIError) {
	workflowDefinition, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}
	refs := collectIssuerCredentialReferences(workflowDefinition)
	if len(refs) == 0 {
		return nil, apierror.New(
			http.StatusBadRequest,
			"credential",
			"pipeline must reference at least one credential-offer credential",
			"no credential-offer credential_id references found",
		)
	}

	filterIssuerID := ""
	if strings.TrimSpace(issuerID) != "" {
		issuerRecord, apiErr := resolveCollectionRecord(app, "credential_issuers", issuerID, "issuer_id")
		if apiErr != nil {
			return nil, apiErr
		}
		filterIssuerID = issuerRecord.Id
	}

	var filtered []issuerCredentialReference
	seen := map[string]struct{}{}
	for _, ref := range refs {
		credentialRecord, apiErr := resolveCollectionRecord(
			app,
			"credentials",
			ref.CredentialID,
			"credential_id",
		)
		if apiErr != nil {
			return nil, apiErr
		}
		if filterIssuerID != "" && credentialRecord.GetString("credential_issuer") != filterIssuerID {
			continue
		}
		normalized := canonify.NormalizePath(ref.CredentialID)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		filtered = append(filtered, issuerCredentialReference{
			StepID:       ref.StepID,
			CredentialID: normalized,
		})
	}
	if len(filtered) == 0 {
		return nil, apierror.New(
			http.StatusBadRequest,
			"issuer_id",
			"no credential_id is associated with issuer_id",
			"no credential-offer step references a credential for the requested issuer",
		)
	}
	return filtered, nil
}

func collectIssuerCredentialReferences(
	workflowDefinition *pipelineinternal.WorkflowDefinition,
) []issuerCredentialReference {
	if workflowDefinition == nil {
		return nil
	}
	var refs []issuerCredentialReference
	collect := func(step pipelineinternal.StepSpec) {
		if step.Use != issuerCICredentialOfferStepUse || step.With.Payload == nil {
			return
		}
		credentialID, ok := step.With.Payload["credential_id"].(string)
		if !ok || strings.TrimSpace(credentialID) == "" {
			return
		}
		refs = append(refs, issuerCredentialReference{
			StepID:       step.ID,
			CredentialID: canonify.NormalizePath(credentialID),
		})
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

func createPipelineRunIssuerTempCredentials(
	e *core.RequestEvent,
	runContext pipelineRunIssuerContext,
) ([]tempCredential, map[string]string, *apierror.APIError) {
	rewriteMap := map[string]string{}
	tempCredentials := []tempCredential{}
	for _, ref := range runContext.credentialRefs {
		credentialRecord, apiErr := resolveCollectionRecord(
			e.App,
			"credentials",
			ref.CredentialID,
			"credential_id",
		)
		if apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return nil, nil, apiErr
		}
		if apiErr := authorizeOwnedOrPublishedRecord(
			credentialRecord,
			runContext.organizationRecord.Id,
			"credentials",
			"credential",
		); apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return nil, nil, apiErr
		}

		rewrittenCredentialYAML, ok := rewriteCredentialStepCIHost(
			credentialRecord.GetString("yaml"),
			runContext.input.IssuerURL,
		)
		if !ok {
			continue
		}
		tempCredential, apiErr := createTempCredential(
			e,
			credentialRecord,
			runContext.organizationRecord.Id,
			runContext.input.CommitSHA,
			rewrittenCredentialYAML,
		)
		if apiErr != nil {
			rollbackPipelineRunIssuerTempCredentials(e, tempCredentials)
			return nil, nil, apiErr
		}
		tempCredentials = append(tempCredentials, tempCredential)
		rewriteMap[canonify.NormalizePath(ref.CredentialID)] = tempCredential.Identifier
	}

	return tempCredentials, rewriteMap, nil
}

func createTempCredential(
	e *core.RequestEvent,
	original *core.Record,
	ownerID string,
	commitSHA string,
	rewrittenYAML string,
) (tempCredential, *apierror.APIError) {
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

	tag := canonify.CanonifyPlain(commitSHA)
	if tag == "" {
		return tempCredential{}, apierror.New(
			http.StatusBadRequest,
			"metadata",
			"commit sha is invalid",
			"commit sha cannot be canonified",
		)
	}
	record.Set("owner", ownerID)
	record.Set("name", uniqueTempCredentialName(e.App, original.GetString("name"), tag, original.GetString("credential_issuer")))
	record.Set("published", false)
	record.Set("yaml", rewrittenYAML)

	if err := e.App.Save(record); err != nil {
		return tempCredential{}, apierror.New(
			http.StatusInternalServerError,
			"credential",
			"failed to create temporary credential",
			err.Error(),
		)
	}
	if len(fileFieldValues) > 0 {
		if err := cloneFiles(e.App, original, record, fileFieldValues); err != nil {
			e.App.Logger().Warn(fmt.Sprintf(
				"failed to clone files for temporary credential %s: %v",
				record.Id,
				err,
			))
		}
	}

	identifier, err := canonify.BuildPath(
		e.App,
		record,
		canonify.CanonifyPaths["credentials"],
		"",
	)
	if err != nil {
		_ = e.App.Delete(record)
		return tempCredential{}, apierror.New(
			http.StatusInternalServerError,
			"credential",
			"failed to build temporary credential identifier",
			err.Error(),
		)
	}

	return tempCredential{Record: record, Identifier: identifier}, nil
}

func uniqueTempCredentialName(app core.App, baseName string, tag string, issuerID string) string {
	base := strings.TrimSpace(baseName)
	if base == "" {
		base = "credential"
	}
	candidateBase := base + "-" + tag
	candidate := candidateBase
	for i := 1; ; i++ {
		_, err := app.FindFirstRecordByFilter(
			"credentials",
			"name = {:name} && credential_issuer = {:issuer}",
			dbx.Params{"name": candidate, "issuer": issuerID},
		)
		if errors.Is(err, sql.ErrNoRows) {
			return candidate
		}
		if err != nil {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", candidateBase, i)
	}
}

func rewriteCredentialStepCIHost(credentialYAML string, issuerURL string) (string, bool) {
	if strings.TrimSpace(credentialYAML) == "" {
		return "", false
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(credentialYAML), &doc); err != nil {
		return "", false
	}
	env, ok := doc["env"].(map[string]any)
	if !ok {
		return "", false
	}
	if _, ok := env["host"].(string); !ok {
		return "", false
	}
	env["host"] = issuerURL
	rewritten, err := yaml.Marshal(doc)
	if err != nil {
		return "", false
	}
	return string(rewritten), true
}

func rewritePipelineRunIssuerYAML(
	pipelineYAML string,
	rewriteMap map[string]string,
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
		rewriteIssuerCredentialStep(&workflowDefinition.Steps[i].StepSpec, rewriteMap)
		for _, onErr := range workflowDefinition.Steps[i].OnError {
			if onErr != nil {
				rewriteIssuerCredentialStep(&onErr.StepSpec, rewriteMap)
			}
		}
		for _, onSuccess := range workflowDefinition.Steps[i].OnSuccess {
			if onSuccess != nil {
				rewriteIssuerCredentialStep(&onSuccess.StepSpec, rewriteMap)
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

func rewriteIssuerCredentialStep(step *pipelineinternal.StepSpec, rewriteMap map[string]string) {
	if step == nil || step.Use != issuerCICredentialOfferStepUse || step.With.Payload == nil {
		return
	}
	credentialID, ok := step.With.Payload["credential_id"].(string)
	if !ok {
		return
	}
	tempIdentifier, ok := rewriteMap[canonify.NormalizePath(credentialID)]
	if !ok {
		return
	}
	step.With.Payload["credential_id"] = tempIdentifier
}

func buildPipelineRunIssuerCleanupMetadata(
	tempCredentials []tempCredential,
) *workflows.MobileRunnerSemaphoreCleanupMetadata {
	if len(tempCredentials) == 0 {
		return nil
	}
	cleanup := &workflows.MobileRunnerSemaphoreCleanupMetadata{}
	for _, credential := range tempCredentials {
		if credential.Record == nil || credential.Record.Id == "" {
			continue
		}
		cleanup.TempCredentials = append(
			cleanup.TempCredentials,
			workflows.MobileRunnerSemaphoreTempCredentialCleanupMetadata{
				RecordID:   credential.Record.Id,
				OwnerID:    credential.Record.GetString("owner"),
				Identifier: credential.Identifier,
			},
		)
	}
	if len(cleanup.TempCredentials) == 0 {
		return nil
	}
	return cleanup
}

func rollbackPipelineRunIssuerTempCredentials(e *core.RequestEvent, tempCredentials []tempCredential) {
	for _, credential := range tempCredentials {
		if credential.Record == nil || credential.Record.Id == "" {
			continue
		}
		if err := e.App.Delete(credential.Record); err != nil {
			e.App.Logger().Warn(fmt.Sprintf(
				"failed to rollback temporary credential %s: %v",
				credential.Record.Id,
				err,
			))
		}
	}
}

func resolvePipelineRunIssuerRunnerID(
	ctx context.Context,
	app core.App,
	workflowDefinition *pipelineinternal.WorkflowDefinition,
	input pipelineRunIssuerRequest,
) (string, bool, bool, *apierror.APIError) {
	hasStepRunner, needsGlobalRunner := pipelineCIMobileRunnerSelectionState(workflowDefinition)
	if strings.TrimSpace(input.RunnerID) != "" {
		return input.RunnerID, hasStepRunner, needsGlobalRunner, nil
	}
	if strings.TrimSpace(input.RunnerType) == "" {
		return "", hasStepRunner, needsGlobalRunner, nil
	}
	runnerID, apiErr := selectPipelineCIRunnerByType(ctx, app, input.RunnerType)
	return runnerID, hasStepRunner, needsGlobalRunner, apiErr
}

func resolveCollectionRecord(
	app core.App,
	collection string,
	identifier string,
	field string,
) (*core.Record, *apierror.APIError) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, apierror.New(
			http.StatusBadRequest,
			field,
			field+" is required",
			"missing "+field,
		)
	}
	record, err := canonify.Resolve(app, identifier)
	if err == nil {
		if record.Collection().Name == collection {
			return record, nil
		}
		directRecord, directErr := app.FindRecordById(collection, identifier)
		if directErr != nil {
			return nil, apierror.New(
				http.StatusBadRequest,
				field,
				field+" is invalid",
				field+" must resolve to a "+collection+" record",
			)
		}
		return directRecord, nil
	}
	record, err = app.FindRecordById(collection, identifier)
	if err != nil {
		return nil, apierror.New(
			http.StatusNotFound,
			field,
			field+" not found",
			err.Error(),
		)
	}
	return record, nil
}

func deleteTempCredentialForOwner(
	app core.App,
	recordID string,
	ownerID string,
) *apierror.APIError {
	record, err := app.FindRecordById("credentials", strings.TrimSpace(recordID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return apierror.New(
			http.StatusInternalServerError,
			"credential",
			"failed to find temporary credential",
			err.Error(),
		)
	}
	if record.GetString("owner") != ownerID {
		return apierror.New(
			http.StatusForbidden,
			"credential",
			"temporary credential owner mismatch",
			"queued cleanup does not belong to the authenticated organization",
		)
	}
	if err := app.Delete(record); err != nil {
		return apierror.New(
			http.StatusInternalServerError,
			"credential",
			"failed to delete temporary credential",
			err.Error(),
		)
	}
	return nil
}
