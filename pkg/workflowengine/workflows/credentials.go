// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for Credentials Issuers.
// It includes the CredentialsIssuersWorkflow, which validates and imports credential issuer metadata.
// The workflow performs various steps including checking the issuer, parsing JSON responses,
// storing credentials, and cleaning up invalid credentials.
package workflows

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CredentialsTaskQueue is the task queue for the credentials workflow.
const (
	CredentialsTaskQueue       = "CredentialsTaskQueue"
	CredentialsIssuerDataQuery = "getCredentialsIssuerData"
	CredentialIssuerSchemaPath = "schemas/credentialissuer/openid-credential-issuer.schema.json"
	CredentialSchemaPath       = "schemas/credentialissuer/credential_config.schema.json"
)

// CredentialsIssuersWorkflow is a workflow that validates and imports credential issuer metadata.
type CredentialsIssuersWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

var credentialsStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

// CredentialsIssuersWorkflowPayload is the payload for the CredentialsIssuersWorkflow.
type CredentialsIssuersWorkflowPayload struct {
	BaseURL  string `json:"base_url"  yaml:"base_url"  validate:"required"`
	IssuerID string `json:"issuer_id" yaml:"issuer_id" validate:"required"`
}

func NewCredentialsIssuersWorkflow() *CredentialsIssuersWorkflow {
	w := &CredentialsIssuersWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

// Name returns the name of the workflow.
func (w *CredentialsIssuersWorkflow) Name() string {
	return "Validate and import Credential Issuer metadata"
}

// GetOptions returns the activity options for the workflow.
func (w *CredentialsIssuersWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *CredentialsIssuersWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// Workflow is the main workflow function for the CredentialsIssuersWorkflow.
// It performs the following steps:
//  1. Executes the CheckCredentialsIssuerActivity to validate the credentials issuer.
//  2. Parses the raw JSON response from the issuer using the JSONActivity.
//  3. Iterates through the credential configurations supported by the issuer and:
//     - Sends each credential to the "store-or-update-extracted-credentials" endpoint.
//     - Logs the stored credentials.
//  4. Returns a WorkflowResult containing a success message and logs.
//
// Parameters:
// - ctx: The workflow context.
// - input: The input for the workflow, containing configuration and payload data.
//
// Returns:
// - workflowengine.WorkflowResult: The result of the workflow execution, including logs.
// - error: An error if any step in the workflow fails.
func (w *CredentialsIssuersWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	credentialsIssuerDataReady := false
	var issuerName, logo string
	var credentialsNumber int

	workflow.SetQueryHandler(ctx, CredentialsIssuerDataQuery, func() (map[string]any, error) {
		if !credentialsIssuerDataReady {
			return nil, workflowengine.NotReadyError{}
		}
		return map[string]any{
			"issuerName":        issuerName,
			"logo":              logo,
			"credentialsNumber": credentialsNumber,
		}, nil
	})
	baseURL, appURL, issuerSchema, issuerID, err := validateInput(input)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	orgID, ok := input.Config["orgID"].(string)
	if !ok || orgID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"orgID",
			input.RunMetadata,
		)
	}

	metadata, err := fetchCredentialIssuerMetadata(ctx, input, baseURL, issuerSchema)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	if len(metadata.CredentialConfigurations) == 0 {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
				"credential issuer metadata contains no credential_configurations_supported entries",
			),
			input.RunMetadata,
		)
	}

	issuerName = metadata.IssuerName
	logo = metadata.Logo
	credentialsNumber = len(metadata.CredentialConfigurations)
	credentialsIssuerDataReady = true

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
		return workflowengine.WorkflowResult{}, err
	}

	return workflowengine.WorkflowResult{
		Message: fmt.Sprintf(
			"Successfully retrieved and stored and update credentials from '%s'",
			metadata.Source,
		),
		Output: map[string]any{
			"issuerName": metadata.IssuerName,
			"logo":       metadata.Logo,
		},
		Log:    storeResult.Logs,
		Errors: metadata.Errors,
	}, nil
}

type credentialIssuerMetadata struct {
	Source                   string
	IssuerName               string
	Logo                     string
	CredentialConfigurations map[string]any
	InvalidCredentials       map[string]bool
	Errors                   map[string][]any
}

type credentialIssuerCredentialStoreParams struct {
	AppURL         string
	IssuerID       string
	OrganizationID string
}

type credentialIssuerCredentialStoreResult struct {
	Logs map[string][]any
}

func fetchCredentialIssuerMetadata(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	baseURL string,
	issuerSchema string,
) (credentialIssuerMetadata, error) {
	logger := workflow.GetLogger(ctx)

	checkIssuer := activities.NewCheckCredentialsIssuerActivity()
	var issuerResult workflowengine.ActivityResult
	err := workflow.ExecuteActivity(ctx, checkIssuer.Name(), workflowengine.ActivityInput{
		Payload: activities.CheckCredentialsIssuerActivityPayload{
			BaseURL: baseURL,
		},
	}).Get(ctx, &issuerResult)
	if err != nil {
		logger.Error("CheckCredentialIssuer failed", "error", err)
		return credentialIssuerMetadata{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	checkIssuerOutput, ok := issuerResult.Output.(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: output", checkIssuer.Name()),
		)
		return credentialIssuerMetadata{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	source, ok := checkIssuerOutput["source"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: source", checkIssuer.Name()),
		)
		return credentialIssuerMetadata{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	rawJSON, ok := checkIssuerOutput["rawJSON"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: rawJSON", checkIssuer.Name()),
		)
		return credentialIssuerMetadata{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}

	parseJSON := activities.NewJSONActivity(
		map[string]reflect.Type{
			"map": reflect.TypeOf(
				map[string]any{},
			),
		},
	)

	errs := make(map[string][]any)
	var result workflowengine.ActivityResult
	invalidCred := make(map[string]bool)
	var issuerName, logo string

	err = workflow.ExecuteActivity(ctx, parseJSON.Name(), workflowengine.ActivityInput{
		Payload: activities.JSONActivityPayload{
			RawJSON:    rawJSON,
			StructType: "map",
		},
	}).Get(ctx, &result)
	if err != nil {
		logger.Error("ParseJSON failed", "error", err)
		return credentialIssuerMetadata{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	issuerData, ok := result.Output.(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: output", parseJSON.Name()),
		)
		return credentialIssuerMetadata{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	validateJSON := activities.NewSchemaValidationActivity()
	validateErr := workflow.ExecuteActivity(ctx, validateJSON.Name(), workflowengine.ActivityInput{
		Payload: activities.SchemaValidationActivityPayload{
			Data:   issuerData,
			Schema: issuerSchema,
		},
	}).Get(ctx, nil)
	issuerLevelValidationErrors := false
	if validateErr != nil {
		details, err := extractAppErrorDetails(validateErr)
		if err != nil {
			return credentialIssuerMetadata{}, workflowengine.NewWorkflowError(
				err,
				input.RunMetadata,
			)
		}

		errs["JSONSchemaValidation"] = details
		issuerLevelValidationErrors = hasIssuerLevelValidationErrors(details)
		invalidCred, err = extractInvalidCredentialsFromErrorDetails(details, input.RunMetadata)
		if err != nil {
			return credentialIssuerMetadata{}, err
		}
	}

	if displayList, ok := issuerData["display"].([]any); ok && len(displayList) > 0 {
		if first, ok := displayList[0].(map[string]any); ok {
			if name, ok := first["name"].(string); ok {
				issuerName = name
			}
			if logoMap, ok := first["logo"].(map[string]any); ok {
				if uri, ok := logoMap["uri"].(string); ok {
					logo = uri
				} else if urlValue, ok := logoMap["url"].(string); ok {
					logo = urlValue
				}
			}
		}
	}

	credConfigs, fallbackInvalidCreds := credentialConfigurationsFromIssuerData(issuerData, errs)
	for credKey := range fallbackInvalidCreds {
		invalidCred[credKey] = true
	}
	if issuerLevelValidationErrors {
		for credKey := range credConfigs {
			invalidCred[credKey] = true
		}
	}
	if source == ".well-known/openid-credential-issuer" {
		if err := validateCredentialIssuerIdentifier(issuerData, baseURL); err != nil {
			errs["CredentialIssuerIdentifier"] = []any{err.Error()}
			for credKey := range credConfigs {
				invalidCred[credKey] = true
			}
		}
	}

	return credentialIssuerMetadata{
		Source:                   source,
		IssuerName:               issuerName,
		Logo:                     logo,
		CredentialConfigurations: credConfigs,
		InvalidCredentials:       invalidCred,
		Errors:                   errs,
	}, nil
}

func storeCredentialIssuerCredentials(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	params credentialIssuerCredentialStoreParams,
	metadata credentialIssuerMetadata,
) (credentialIssuerCredentialStoreResult, error) {
	if params.IssuerID == "" {
		return credentialIssuerCredentialStoreResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
				"credential issuer record id is required",
			),
			input.RunMetadata,
		)
	}

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	logs := make(map[string][]any)
	for credKey, credential := range metadata.CredentialConfigurations {
		conformant := true
		if metadata.InvalidCredentials[credKey] {
			conformant = false
		}

		storeInput := workflowengine.ActivityInput{
			Payload: activities.InternalHTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					params.AppURL,
					"api", "credentials_issuers", "store-or-update-extracted-credentials"),
				Body: map[string]any{
					"issuerID":   params.IssuerID,
					"credKey":    credKey,
					"credential": credential,
					"conformant": conformant,
					"orgID":      params.OrganizationID,
				},
				ExpectedStatus: 200,
			},
		}
		var storeResponse workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(ctx, internalHTTPActivity.Name(), storeInput).
			Get(ctx, &storeResponse); err != nil {
			return credentialIssuerCredentialStoreResult{Logs: logs}, err
		}
		key, ok := storeResponse.Output.(map[string]any)["body"].(map[string]any)["key"]
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: body.key", internalHTTPActivity.Name()),
			)
			return credentialIssuerCredentialStoreResult{}, workflowengine.NewWorkflowError(
				appErr,
				input.RunMetadata,
			)
		}

		logs["StoredCredentials"] = append(
			logs["StoredCredentials"],
			key,
		)
	}

	return credentialIssuerCredentialStoreResult{
		Logs: logs,
	}, nil
}

// Start initializes and starts the CredentialsIssuersWorkflow execution.
// It loads environment variables, configures the Temporal client with the specified namespace,
// and sets up workflow options including a unique workflow ID and optional memo.
// The workflow is then executed with the provided input.
//
// Parameters:
//   - input: A WorkflowInput object containing configuration and input data for the workflow.
//
// Returns:
//   - result: A WorkflowResult object (empty in this implementation).
//   - err: An error if the workflow fails to start or if there is an issue with the Temporal client.
//
// Errors:
//   - Returns an error if the Temporal client cannot be created or if the workflow execution fails.
func (w *CredentialsIssuersWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "Credentials-Workflow-" + uuid.NewString(),
		TaskQueue:                CredentialsTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	return credentialsStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

// GetCredentialOfferWorkflow is a workflow that gets a credential offer from a stored credential.
type GetCredentialOfferWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type GetCredentialOfferWorkflowPayload struct {
	CredentialID string `json:"credential_id" yaml:"credential_id" validate:"required"`
}

func NewGetCredentialOfferWorkflow() *GetCredentialOfferWorkflow {
	w := &GetCredentialOfferWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}
func (w *GetCredentialOfferWorkflow) Name() string {
	return "Get a credential offer"
}

// GetOptions returns the activity options for the workflow.
func (w *GetCredentialOfferWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

// Workflow is the main workflow function for the GetCredentialOfferWorkflow.
// It executes the following steps:
//  1. Executes the CheckCredentialsIssuerActivity to validate the credentials issuer.
//  2. Parses the raw JSON response from the issuer using the JSONActivity.
//  3. Iterates through the credential configurations supported by the issuer and:
//     - Sends each credential to the "store-or-update-extracted-credentials" endpoint.
//     - Logs the stored credentials.
//  4. Returns a WorkflowResult containing a success message and logs.
//
// Parameters:
// - ctx: The workflow context.
// - input: The input for the workflow, containing configuration and payload data.
//
// Returns:
// - workflowengine.WorkflowResult: The result of the workflow execution, including logs.
// - error: An error if any step in the workflow fails.
func (w *GetCredentialOfferWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *GetCredentialOfferWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

	payload, err := workflowengine.DecodePayload[GetCredentialOfferWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}
	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}
	act := activities.NewInternalHTTPActivity()
	var result workflowengine.ActivityResult
	request := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				appURL,
				"api", "credential", "get-credential-offer",
			),
			QueryParams: map[string]string{
				"credential_identifier": payload.CredentialID,
			},
			ExpectedStatus: 200,
		},
	}
	err = workflow.ExecuteActivity(ctx, act.Name(), request).Get(ctx, &result)
	if err != nil {
		logger.Error("HTTPActivity failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
	responseBody, ok := result.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"output is not a map",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}
	dynamic, ok := responseBody["dynamic"].(bool)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"dynamic is not a bool",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}
	if !dynamic {
		credentialOffer, ok := responseBody["credential_offer"].(string)
		if !ok {
			wErr := workflowengine.NewAppError(
				errCode,
				"credential_offer is not a string",
				result.Output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				wErr,
				input.RunMetadata,
			)
		}

		return workflowengine.WorkflowResult{
			Message: "Successfully retrieved credential offer",
			Output:  credentialOffer,
		}, nil
	}
	code, ok := responseBody["code"].(string)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"yaml code is not a string",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	var stepCIResult workflowengine.ActivityResult
	stepCIInput := workflowengine.ActivityInput{
		Payload: activities.StepCIWorkflowActivityPayload{
			Yaml: code,
		},
	}
	err = workflow.ExecuteActivity(ctx, stepCIActivity.Name(), stepCIInput).Get(ctx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIActivity failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	captures, ok := stepCIResult.Output.(map[string]any)["captures"].(map[string]any)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"captures is not a map",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}
	credentialOffer, ok := captures["deeplink"].(string)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"deeplink missing or invalid from captures",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}
	return workflowengine.WorkflowResult{
		Message: "Successfully retrieved credential offer",
		Output:  credentialOffer,
	}, nil
}

func extractInvalidCredentialsFromErrorDetails(
	details []any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
) (map[string]bool, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityErrorDetails]
	invalidCred := map[string]bool{}

	rawMap, ok := details[0].(map[string]any)
	if !ok {
		wErr := workflowengine.NewAppError(errCode, "details[0] is not a map")
		return nil, workflowengine.NewWorkflowError(wErr, runMetadata)
	}

	causes, ok := rawMap["Causes"].([]any)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"details should contain causes from validation error",
		)
		return nil, workflowengine.NewWorkflowError(wErr, runMetadata)
	}

	for _, cause := range causes {
		causeMap, ok := cause.(map[string]any)
		if !ok {
			wErr := workflowengine.NewAppError(
				errCode,
				"each cause should be a map",
			)
			return nil, workflowengine.NewWorkflowError(wErr, runMetadata)
		}

		instanceLocationStr := workflowengine.AsSliceOfStrings(causeMap["InstanceLocation"])
		if len(instanceLocationStr) > 1 &&
			instanceLocationStr[0] == "credential_configurations_supported" {
			invalidCred[instanceLocationStr[1]] = true
		}
	}

	return invalidCred, nil
}

func credentialConfigurationsFromIssuerData(
	issuerData map[string]any,
	errs map[string][]any,
) (map[string]any, map[string]bool) {
	credConfigs := map[string]any{}
	invalidCreds := map[string]bool{}

	rawCredConfigs, exists := issuerData["credential_configurations_supported"]
	if typed, ok := rawCredConfigs.(map[string]any); ok && len(typed) > 0 {
		return typed, invalidCreds
	}

	legacyConfigs := legacyCredentialConfigurations(issuerData["credentials_supported"])
	if len(legacyConfigs) > 0 {
		for key := range legacyConfigs {
			invalidCreds[key] = true
		}
		errs["LegacyCredentialsSupportedFallback"] = []any{
			"credential_configurations_supported is missing, null, empty, or invalid; imported legacy credentials_supported entries as non-conformant credentials",
		}
		return legacyConfigs, invalidCreds
	}

	if exists && rawCredConfigs != nil {
		errs["InvalidCredentialConfigurations"] = []any{
			fmt.Sprintf(
				"credential_configurations_supported must be an object, got %T",
				rawCredConfigs,
			),
		}
	}
	errs["NoCredentialConfigurations"] = []any{
		"credential_configurations_supported is missing, null, empty, or invalid",
	}

	return credConfigs, invalidCreds
}

func legacyCredentialConfigurations(raw any) map[string]any {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	configs := make(map[string]any, len(items))
	for index, item := range items {
		credential, ok := item.(map[string]any)
		if !ok {
			continue
		}

		key := legacyCredentialConfigurationKey(credential, index)
		configs[key] = credential
	}

	return configs
}

func legacyCredentialConfigurationKey(credential map[string]any, index int) string {
	for _, field := range []string{"id", "credential_configuration_id", "scope"} {
		if value, ok := credential[field].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}

	if types, ok := credential["types"].([]any); ok && len(types) > 0 {
		if lastType, ok := types[len(types)-1].(string); ok && strings.TrimSpace(lastType) != "" {
			return lastType
		}
	}

	return fmt.Sprintf("legacy-credential-%d", index+1)
}

func hasIssuerLevelValidationErrors(details []any) bool {
	rawMap, ok := details[0].(map[string]any)
	if !ok {
		return true
	}

	causes, ok := rawMap["Causes"].([]any)
	if !ok {
		return true
	}

	for _, cause := range causes {
		causeMap, ok := cause.(map[string]any)
		if !ok {
			return true
		}
		instanceLocationStr := workflowengine.AsSliceOfStrings(causeMap["InstanceLocation"])
		if len(instanceLocationStr) == 0 ||
			instanceLocationStr[0] != "credential_configurations_supported" {
			return true
		}
	}

	return false
}

func validateCredentialIssuerIdentifier(
	issuerData map[string]any,
	baseURL string,
) error {
	metadataIssuer, ok := issuerData["credential_issuer"].(string)
	if !ok || strings.TrimSpace(metadataIssuer) == "" {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.SchemaValidationFailed],
			"credential_issuer is required",
		)
	}

	expectedIssuer := credentialIssuerIdentifierFromInput(baseURL)
	if expectedIssuer == "" {
		return nil
	}
	if normalizeCredentialIssuerIdentifier(metadataIssuer) !=
		normalizeCredentialIssuerIdentifier(expectedIssuer) {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.SchemaValidationFailed],
			fmt.Sprintf(
				"credential_issuer %q does not match expected issuer identifier %q",
				metadataIssuer,
				expectedIssuer,
			),
		)
	}

	return nil
}

func credentialIssuerIdentifierFromInput(rawURL string) string {
	const wellKnownPath = "/.well-known/openid-credential-issuer"

	cleanURL := strings.TrimSpace(rawURL)
	if cleanURL == "" {
		return ""
	}
	if !strings.HasPrefix(cleanURL, "https://") && !strings.HasPrefix(cleanURL, "http://") {
		cleanURL = "https://" + cleanURL
	}

	parsed, err := url.Parse(strings.TrimRight(cleanURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	switch {
	case parsed.Path == wellKnownPath:
		parsed.Path = ""
	case strings.HasPrefix(parsed.Path, wellKnownPath+"/"):
		parsed.Path = strings.TrimPrefix(parsed.Path, wellKnownPath)
	case strings.HasSuffix(parsed.Path, wellKnownPath):
		parsed.Path = strings.TrimSuffix(parsed.Path, wellKnownPath)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func normalizeCredentialIssuerIdentifier(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(strings.TrimSpace(rawURL), "/")
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func extractAppErrorDetails(err error) ([]any, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityError]
	var actErr *temporal.ActivityError
	if errors.As(err, &actErr) {
		var appErr *temporal.ApplicationError
		if errors.As(actErr.Unwrap(), &appErr) {
			var details []any
			derr := appErr.Details(&details)
			if derr == nil {
				return details, nil
			}
			return nil, workflowengine.NewAppError(errCode, derr.Error())
		}
		return nil, workflowengine.NewAppError(errCode, actErr.Unwrap().Error())
	}
	return nil, workflowengine.NewAppError(errCode, err.Error())
}

func validateInput(
	input workflowengine.WorkflowInput,
) (baseURL, appURL, issuerSchema, issuerID string, err error) {
	payload, err := workflowengine.DecodePayload[CredentialsIssuersWorkflowPayload](input.Payload)
	if err != nil {
		return "", "", "", "", workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return "", "", "", "", workflowengine.NewMissingConfigError("app_url", input.RunMetadata)
	}
	issuerSchema, ok = input.Config["issuer_schema"].(string)
	if !ok || issuerSchema == "" {
		return "", "", "", "", workflowengine.NewMissingConfigError(
			"issuer_schema",
			input.RunMetadata,
		)
	}

	return payload.BaseURL, appURL, issuerSchema, payload.IssuerID, nil
}
