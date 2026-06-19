// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const issuerCICredentialOfferStepUse = "credential-offer"
const issuerCITempCredentialsConfigKey = "temp_credentials"
const credentialResourceDomain = "credential"
const issuerCIURLField = "issuer_url"

type PipelineRunIssuerResponse struct {
	PipelineQueueResponse
	TempCredentials    []TempCredentialResponse `json:"temp_credentials,omitempty"`
	PipelineIdentifier string                   `json:"pipeline_identifier,omitempty"`
	Warning            string                   `json:"warning,omitempty"`
}

type TempCredentialResponse struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
}

type tempCredential = pipelineCITempRecordResult

func HandlePipelineRunIssuer() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, apiErr := parsePipelineCIBaseRequest(e, "credential_ids", issuerCIURLField)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := validatePipelineCIBaseRequest(input, issuerCIURLField); apiErr != nil {
			return apiErr
		}

		runContext, apiErr := resolvePipelineCIRunContext(e, input.PipelineIdentifier)
		if apiErr != nil {
			return apiErr
		}
		credentialRefs, apiErr := resolvePipelineRunIssuerCredentialReferences(
			e.App,
			runContext.PipelineYAML,
			input.IDs,
		)
		if apiErr != nil {
			return apiErr
		}
		tempCredentials, rewriteMap, apiErr := createPipelineCITempRecords(
			e,
			pipelineCITempRecordsOptions{
				Refs:           credentialRefs,
				Collection:     "credentials",
				IdentifierKey:  "credential_id",
				IdentifierName: credentialResourceDomain,
				ResourceDomain: credentialResourceDomain,
				ResourceName:   credentialResourceDomain,
				OwnerID:        runContext.OrganizationRecord.Id,
				CommitSHA:      input.CommitSHA,
				HostURL:        input.HostURL,
				RelationField:  "credential_issuer",
				UniqueName:     uniqueTempCredentialName,
			},
		)
		if apiErr != nil {
			return apiErr
		}
		rewrittenYAML, apiErr := rewritePipelineRunIssuerYAML(runContext.PipelineYAML, rewriteMap)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempCredentials, credentialResourceDomain)
			return apiErr
		}
		workflowDefinition, apiErr := parsePipelineCIWorkflow(rewrittenYAML)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempCredentials, credentialResourceDomain)
			return apiErr
		}
		runnerID, hasStepRunner, needsGlobalRunner, apiErr := resolvePipelineCIRunnerID(
			e.Request.Context(),
			e.App,
			runContext.OrganizationRecord.Id,
			workflowDefinition,
			input,
		)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempCredentials, credentialResourceDomain)
			return apiErr
		}
		warning := pipelineCIIgnoredRunnerWarning(
			input.RunnerID,
			input.RunnerType,
			hasStepRunner,
			needsGlobalRunner,
		)
		manipulatedYAML, apiErr := injectPipelineCIGlobalRunnerID(
			rewrittenYAML,
			workflowDefinition,
			runnerID,
			hasStepRunner,
			needsGlobalRunner,
		)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempCredentials, credentialResourceDomain)
			return apiErr
		}
		notification := buildPipelineGitHubPRNotification(
			input.Metadata,
			e.App.Settings().Meta.AppURL,
			input.PipelineIdentifier,
			runnerID,
			resolveWalletAPKGitHubPRRunnerType(e.App, runnerID, input.RunnerType),
			activities.GitHubPRCommentSectionIssuer,
		)

		queueResponse, apiErr := enqueuePipelineRun(e, pipelineQueueRunContext{
			pipelineRecord:     runContext.PipelineRecord,
			pipelineIdentifier: input.PipelineIdentifier,
			organizationRecord: runContext.OrganizationRecord,
			userID:             runContext.UserID,
			userName:           runContext.UserName,
			userEmail:          runContext.UserEmail,
			yaml:               manipulatedYAML,
			metadata:           input.Metadata,
			runType:            pipelineinternal.RunTypeCI,
			cleanup:            buildPipelineRunIssuerCleanupMetadata(tempCredentials),
			notification:       notification,
		})
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempCredentials, credentialResourceDomain)
			return apiErr
		}

		response := buildPipelineRunIssuerResponse(
			queueResponse,
			tempCredentials,
			input.PipelineIdentifier,
			warning,
		)
		if err := maybeCreatePipelineGitHubPRComment(
			e.Request.Context(),
			notification,
			response.PipelineQueueResponse,
		); err != nil {
			e.App.Logger().Warn(fmt.Sprintf(
				"failed to create github pr comment for issuer pipeline run %s/%s: %v",
				response.WorkflowID,
				response.RunID,
				err,
			))
		}

		return e.JSON(http.StatusOK, response)
	}
}

func buildPipelineRunIssuerResponse(
	queueResponse PipelineQueueResponse,
	tempCredentials []tempCredential,
	pipelineIdentifier string,
	warning string,
) PipelineRunIssuerResponse {
	if queueResponse.Position != nil {
		position := *queueResponse.Position + 1
		queueResponse.Position = &position
	}
	response := PipelineRunIssuerResponse{
		PipelineQueueResponse: queueResponse,
		PipelineIdentifier:    pipelineIdentifier,
		Warning:               warning,
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

func uniqueTempCredentialName(app core.App, baseName string, tag string, issuerID string) string {
	base := strings.TrimSpace(baseName)
	if base == "" {
		base = credentialResourceDomain
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

func resolvePipelineRunIssuerCredentialReferences(
	app core.App,
	pipelineYAML string,
	credentialIDs []string,
) ([]string, *apierror.APIError) {
	workflowDefinition, apiErr := parsePipelineCIWorkflow(pipelineYAML)
	if apiErr != nil {
		return nil, apiErr
	}
	refs := collectPipelineCIReferences(
		workflowDefinition,
		issuerCICredentialOfferStepUse,
		"credential_id",
	)
	if len(refs) == 0 {
		return nil, apierror.New(
			http.StatusBadRequest,
			credentialResourceDomain,
			"pipeline must reference at least one credential-offer credential",
			"no credential-offer credential_id references found",
		)
	}

	requested, apiErr := resolvePipelineCIReferenceFilter(
		app,
		"credentials",
		credentialIDs,
		"credential_ids",
		credentialResourceDomain,
	)
	if apiErr != nil {
		return nil, apiErr
	}

	var filtered []string
	seen := map[string]struct{}{}
	matched := map[string]struct{}{}
	for _, ref := range refs {
		if _, apiErr := resolveCanonicalCollectionRecord(
			app,
			"credentials",
			ref,
			"credential_id",
			credentialResourceDomain,
		); apiErr != nil {
			return nil, apiErr
		}
		normalized := canonify.NormalizePath(ref)
		if len(requested) > 0 {
			if _, ok := requested[normalized]; !ok {
				continue
			}
			matched[normalized] = struct{}{}
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		filtered = append(filtered, normalized)
	}
	if len(filtered) == 0 {
		return nil, apierror.New(
			http.StatusBadRequest,
			"credential_ids",
			"no credential_id matches credential_ids",
			"no credential-offer step references a requested credential_id",
		)
	}
	if len(requested) > len(matched) {
		return nil, apierror.New(
			http.StatusBadRequest,
			"credential_ids",
			"credential_ids must be referenced by the pipeline",
			"one or more requested credential_ids are not used by credential-offer steps",
		)
	}
	return filtered, nil
}

func rewritePipelineRunIssuerYAML(
	pipelineYAML string,
	rewriteMap map[string]string,
) (string, *apierror.APIError) {
	return rewritePipelineCIStepRefsYAML(
		pipelineYAML,
		rewriteMap,
		issuerCICredentialOfferStepUse,
		"credential_id",
	)
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

func resolveCanonicalCollectionRecord(
	app core.App,
	collection string,
	identifier string,
	field string,
	resourceName string,
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
	if err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			field,
			field+" is invalid",
			field+" must be a canonical "+resourceName+" identifier",
		)
	}
	if record.Collection().Name != collection {
		return nil, apierror.New(
			http.StatusBadRequest,
			field,
			field+" is invalid",
			field+" must resolve to a "+resourceName+" record",
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
			credentialResourceDomain,
			"failed to find temporary credential",
			err.Error(),
		)
	}
	if record.GetString("owner") != ownerID {
		return apierror.New(
			http.StatusForbidden,
			credentialResourceDomain,
			"temporary credential owner mismatch",
			"queued cleanup does not belong to the authenticated organization",
		)
	}
	if err := app.Delete(record); err != nil {
		return apierror.New(
			http.StatusInternalServerError,
			credentialResourceDomain,
			"failed to delete temporary credential",
			err.Error(),
		)
	}
	return nil
}
