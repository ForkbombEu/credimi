// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	FidesCredentialIssuersTaskQueue    = "FidesCredentialIssuersTaskQueue"
	FidesCredentialIssuersWorkflowName = "Import Fides Credential Issuers"
)

type FidesCredentialIssuersWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

var fidesCredentialIssuersStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

func NewFidesCredentialIssuersWorkflow() *FidesCredentialIssuersWorkflow {
	w := &FidesCredentialIssuersWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (w *FidesCredentialIssuersWorkflow) Name() string {
	return FidesCredentialIssuersWorkflowName
}

func (w *FidesCredentialIssuersWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *FidesCredentialIssuersWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *FidesCredentialIssuersWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "Fides-Credential-Issuers-" + uuid.NewString(),
		TaskQueue:                FidesCredentialIssuersTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	return fidesCredentialIssuersStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

func (w *FidesCredentialIssuersWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}
	issuerSchema, ok := input.Config["issuer_schema"].(string)
	if !ok || issuerSchema == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"issuer_schema",
			input.RunMetadata,
		)
	}
	orgID, ok := input.Config["orgID"].(string)
	if !ok || orgID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"orgID",
			input.RunMetadata,
		)
	}

	issuers, err := fetchFidesCredentialIssuerURLs(ctx, input)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	if len(issuers) == 0 {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
				"Fides catalog returned no credential issuers",
			),
			input.RunMetadata,
		)
	}

	var imported []string
	logs := map[string][]any{}
	errs := map[string]any{}
	for _, issuerURL := range issuers {
		metadata, err := fetchCredentialIssuerMetadata(ctx, input, issuerURL, issuerSchema)
		if err != nil {
			errs[issuerURL] = err.Error()
			continue
		}

		issuerID, err := storeOrUpdateCredentialIssuerRecord(
			ctx,
			input,
			appURL,
			issuerURL,
			orgID,
			metadata.IssuerName,
			metadata.Logo,
		)
		if err != nil {
			errs[issuerURL] = err.Error()
			continue
		}

		storeLogs := map[string][]any{}
		if len(metadata.CredentialConfigurations) == 0 {
			storeLogs["NoCredentialConfigurations"] = []any{
				"credential issuer imported without credentials",
			}
		} else {
			storeResult, err := storeCredentialIssuerCredentials(
				ctx,
				input,
				credentialIssuerCredentialStoreParams{
					AppURL:         appURL,
					IssuerID:       issuerID,
					OrganizationID: orgID,
				},
				metadata,
			)
			if err != nil {
				errs[issuerURL] = err.Error()
				continue
			}
			storeLogs = storeResult.Logs
		}

		imported = append(imported, issuerURL)
		logs[issuerURL] = []any{storeLogs}
		if len(metadata.Errors) > 0 {
			errs[issuerURL] = metadata.Errors
		}
	}

	return workflowengine.WorkflowResult{
		Message: fmt.Sprintf(
			"Imported %d credential issuers from Fides with %d issuer errors",
			len(imported),
			len(errs),
		),
		Output: map[string]any{
			"issuers": imported,
		},
		Log:    logs,
		Errors: errs,
	}, nil
}

func fetchFidesCredentialIssuerURLs(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) ([]string, error) {
	httpActivity := activities.NewHTTPActivity()
	parseActivity := activities.NewParseFidesCredentialIssuersActivity()

	var issuers []string
	page := 0
	for {
		url := activities.FidesCredentialIssuersURL
		if page > 0 {
			url = fmt.Sprintf("%s?page=%d", activities.FidesCredentialIssuersURL, page)
		}

		var httpResult workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method:         http.MethodGet,
				URL:            url,
				ExpectedStatus: http.StatusOK,
			},
		}).Get(ctx, &httpResult); err != nil {
			return nil, workflowengine.NewWorkflowError(err, input.RunMetadata)
		}

		body, ok := httpResult.Output.(map[string]any)["body"]
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: body", httpActivity.Name()),
			)
			return nil, workflowengine.NewWorkflowError(appErr, input.RunMetadata)
		}

		var parseResult workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(ctx, parseActivity.Name(), workflowengine.ActivityInput{
			Payload: activities.ParseFidesCredentialIssuersActivityPayload{Data: body},
		}).Get(ctx, &parseResult); err != nil {
			return nil, workflowengine.NewWorkflowError(err, input.RunMetadata)
		}

		pageResult, ok := fidesCredentialIssuersFromActivityOutput(parseResult.Output)
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: output", parseActivity.Name()),
			)
			return nil, workflowengine.NewWorkflowError(appErr, input.RunMetadata)
		}
		issuers = append(issuers, pageResult.Issuers...)

		if pageResult.TotalPages == 0 || pageResult.PageNumber >= pageResult.TotalPages-1 {
			break
		}
		page++
	}

	return issuers, nil
}

func storeOrUpdateCredentialIssuerRecord(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	appURL string,
	issuerURL string,
	orgID string,
	name string,
	logo string,
) (string, error) {
	internalHTTPActivity := activities.NewInternalHTTPActivity()
	var storeResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, internalHTTPActivity.Name(), workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodPost,
			URL: utils.JoinURL(
				appURL,
				"api", "credentials_issuers", "store-or-update",
			),
			Body: map[string]any{
				"url":   issuerURL,
				"orgID": orgID,
				"name":  name,
				"logo":  logo,
			},
			ExpectedStatus: http.StatusOK,
		},
	}).Get(ctx, &storeResult); err != nil {
		return "", workflowengine.NewWorkflowError(err, input.RunMetadata)
	}

	body, ok := storeResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: body", internalHTTPActivity.Name()),
		)
		return "", workflowengine.NewWorkflowError(appErr, input.RunMetadata)
	}
	record, ok := body["record"].(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: body.record", internalHTTPActivity.Name()),
		)
		return "", workflowengine.NewWorkflowError(appErr, input.RunMetadata)
	}
	id, ok := record["id"].(string)
	if !ok || id == "" {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: body.record.id", internalHTTPActivity.Name()),
		)
		return "", workflowengine.NewWorkflowError(appErr, input.RunMetadata)
	}

	return id, nil
}

func fidesCredentialIssuersFromActivityOutput(
	output any,
) (activities.ParseFidesCredentialIssuersActivityResponse, bool) {
	response, err := workflowengine.DecodePayload[activities.ParseFidesCredentialIssuersActivityResponse](
		output,
	)
	if err != nil {
		return activities.ParseFidesCredentialIssuersActivityResponse{}, false
	}
	return response, true
}
