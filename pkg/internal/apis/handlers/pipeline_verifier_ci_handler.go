// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
)

const verifierCIUseCaseVerificationStepUse = "use-case-verification-deeplink"
const verifierCITempUseCasesConfigKey = "temp_use_case_verifications"
const useCaseVerificationResourceDomain = "use_case_verification"

type PipelineRunVerifierResponse struct {
	PipelineQueueResponse
	TempUseCases       []TempUseCaseVerificationResponse `json:"temp_use_cases,omitempty"`
	PipelineIdentifier string                            `json:"pipeline_identifier,omitempty"`
	Warning            string                            `json:"warning,omitempty"`
}

type TempUseCaseVerificationResponse struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
}

type tempUseCaseVerification = pipelineCITempRecordResult

func HandlePipelineRunVerifier() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, apiErr := parsePipelineCIBaseRequest(e, "use_case_ids", "verifier_url")
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		if apiErr := validatePipelineCIBaseRequest(input, "verifier_url"); apiErr != nil {
			return apiErr.JSON(e)
		}

		runContext, apiErr := resolvePipelineCIRunContext(e, input.PipelineIdentifier)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		useCaseRefs, apiErr := resolvePipelineRunVerifierUseCaseReferences(
			e.App,
			runContext.PipelineYAML,
			input.IDs,
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		tempUseCases, rewriteMap, apiErr := createPipelineCITempRecords(
			e,
			pipelineCITempRecordsOptions{
				Refs:           useCaseRefs,
				Collection:     "use_cases_verifications",
				IdentifierKey:  "use_case_id",
				IdentifierName: "use case verification",
				ResourceDomain: useCaseVerificationResourceDomain,
				ResourceName:   "use case verification",
				OwnerID:        runContext.OrganizationRecord.Id,
				CommitSHA:      input.CommitSHA,
				HostURL:        input.HostURL,
			},
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		rewrittenYAML, apiErr := rewritePipelineCIStepRefsYAML(
			runContext.PipelineYAML,
			rewriteMap,
			verifierCIUseCaseVerificationStepUse,
			"use_case_id",
		)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempUseCases, "use case verification")
			return apiErr.JSON(e)
		}
		workflowDefinition, apiErr := parsePipelineCIWorkflow(rewrittenYAML)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempUseCases, "use case verification")
			return apiErr.JSON(e)
		}
		runnerID, hasStepRunner, needsGlobalRunner, apiErr := resolvePipelineCIRunnerID(
			e.Request.Context(),
			e.App,
			runContext.OrganizationRecord.Id,
			workflowDefinition,
			input,
		)
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempUseCases, "use case verification")
			return apiErr.JSON(e)
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
			rollbackPipelineCITempRecords(e, tempUseCases, "use case verification")
			return apiErr.JSON(e)
		}
		notification := buildPipelineGitHubPRNotification(
			input.Metadata,
			e.App.Settings().Meta.AppURL,
			input.PipelineIdentifier,
			runnerID,
			resolveWalletAPKGitHubPRRunnerType(e.App, runnerID, input.RunnerType),
			activities.GitHubPRCommentSectionVerifier,
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
			cleanup:            buildPipelineRunVerifierCleanupMetadata(tempUseCases),
			notification:       notification,
		})
		if apiErr != nil {
			rollbackPipelineCITempRecords(e, tempUseCases, "use case verification")
			return apiErr.JSON(e)
		}

		response := buildPipelineRunVerifierResponse(
			queueResponse,
			tempUseCases,
			input.PipelineIdentifier,
			warning,
		)
		if err := maybeCreatePipelineGitHubPRComment(
			e.Request.Context(),
			notification,
			response.PipelineQueueResponse,
		); err != nil {
			e.App.Logger().Warn(fmt.Sprintf(
				"failed to create github pr comment for verifier pipeline run %s/%s: %v",
				response.WorkflowID,
				response.RunID,
				err,
			))
		}

		return e.JSON(http.StatusOK, response)
	}
}

func buildPipelineRunVerifierResponse(
	queueResponse PipelineQueueResponse,
	tempUseCases []tempUseCaseVerification,
	pipelineIdentifier string,
	warning string,
) PipelineRunVerifierResponse {
	if queueResponse.Position != nil {
		position := *queueResponse.Position + 1
		queueResponse.Position = &position
	}
	response := PipelineRunVerifierResponse{
		PipelineQueueResponse: queueResponse,
		PipelineIdentifier:    pipelineIdentifier,
		Warning:               warning,
	}
	for _, useCase := range tempUseCases {
		if useCase.Record == nil {
			continue
		}
		response.TempUseCases = append(response.TempUseCases, TempUseCaseVerificationResponse{
			ID:         useCase.Record.Id,
			Identifier: useCase.Identifier,
		})
	}
	return response
}

func resolvePipelineRunVerifierUseCaseReferences(
	app core.App,
	pipelineYAML string,
	useCaseIDs []string,
) ([]string, *apierror.APIError) {
	workflowDefinition, apiErr := parsePipelineCIWorkflow(pipelineYAML)
	if apiErr != nil {
		return nil, apiErr
	}
	refs := collectPipelineCIReferences(
		workflowDefinition,
		verifierCIUseCaseVerificationStepUse,
		"use_case_id",
	)
	if len(refs) == 0 {
		return nil, apierror.New(
			http.StatusBadRequest,
			useCaseVerificationResourceDomain,
			"pipeline must reference at least one use-case-verification-deeplink use case",
			"no use-case-verification-deeplink use_case_id references found",
		)
	}

	requested, apiErr := resolvePipelineCIReferenceFilter(
		app,
		"use_cases_verifications",
		useCaseIDs,
		"use_case_ids",
		"use case verification",
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
			"use_cases_verifications",
			ref,
			"use_case_id",
			"use case verification",
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
			"use_case_ids",
			"no use_case_id matches use_case_ids",
			"no use-case-verification-deeplink step references a requested use_case_id",
		)
	}
	if len(requested) > len(matched) {
		return nil, apierror.New(
			http.StatusBadRequest,
			"use_case_ids",
			"use_case_ids must be referenced by the pipeline",
			"one or more requested use_case_ids are not used by use-case-verification-deeplink steps",
		)
	}
	return filtered, nil
}

func buildPipelineRunVerifierCleanupMetadata(
	tempUseCases []tempUseCaseVerification,
) *workflows.MobileRunnerSemaphoreCleanupMetadata {
	if len(tempUseCases) == 0 {
		return nil
	}
	cleanup := &workflows.MobileRunnerSemaphoreCleanupMetadata{}
	for _, useCase := range tempUseCases {
		if useCase.Record == nil || useCase.Record.Id == "" {
			continue
		}
		cleanup.TempUseCaseVerifications = append(
			cleanup.TempUseCaseVerifications,
			workflows.MobileRunnerSemaphoreTempCredentialCleanupMetadata{
				RecordID:   useCase.Record.Id,
				OwnerID:    useCase.Record.GetString("owner"),
				Identifier: useCase.Identifier,
			},
		)
	}
	if len(cleanup.TempUseCaseVerifications) == 0 {
		return nil
	}
	return cleanup
}
