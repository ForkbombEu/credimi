// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	engine "github.com/forkbombeu/credimi/pkg/templateengine"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

type Variable struct {
	FieldName string      `json:"field_name" validate:"required"`
	Value     interface{} `json:"value"      validate:"required"`
	CredimiID string      `json:"credimi_id" validate:"required"`
}

type SaveVariablesAndStartRequestInput struct {
	ConfigsWithFields map[string][]Variable `json:"configs_with_fields" validate:"required"`
	ConfigsWithJSON   map[string]string     `json:"configs_with_json"   validate:"required"`
}

type openID4VPTestInputFile struct {
	Variant  json.RawMessage `json:"variant" yaml:"variant" validate:"required,oneof=json variables yaml"`
	Form     workflows.Form  `json:"form"    yaml:"form"`
	TestName string          `json:"test"    yaml:"test"    validate:"required"`
}
type vLEICheckInput struct {
	CredentialID string `json:"credentialID"`
	ServerURL    string `json:"serverURL"`
}

type EudiwInput struct {
	Nonce string `json:"nonce" yaml:"nonce" validate:"required"`
	ID    string `json:"id"    yaml:"id"    validate:"required"`
}

type Author string

type WorkflowStarterParams struct {
	App       core.App
	YAMLData  string
	Email     string
	Namespace string
	Memo      map[string]interface{}
	Author    Author
	TestName  string
	Protocol  string
	Version   string
	UserName  string
}

type WorkflowStarter func(params WorkflowStarterParams) (workflowengine.WorkflowResult, error)

func conformanceInputParameters(
	raw map[string]any,
	excluded map[string]struct{},
) map[string]any {
	parameters := map[string]any{}
	for key, value := range raw {
		if _, skip := excluded[key]; skip {
			continue
		}
		parameters[key] = value
	}
	return parameters
}

func conformanceInputString(raw map[string]any, key string) string {
	if v, ok := raw[key].(string); ok {
		return v
	}
	return ""
}

var workflowRegistry = map[Author]WorkflowStarter{
	Author(workflows.EWCSuite):               startEWCWorkflow,
	Author(workflows.WebuildSuite):           startWebuildWorkflow,
	Author(workflows.OpenIDConformanceSuite): startOpenID4VPWalletWorkflow,
	Author(workflows.EudiwSuite):             startEudiwWorkflow,
	Author(workflows.VLEISuite):              startvLEIWorkflow,
}

var (
	openID4VPWalletWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewOpenID4VPWalletWorkflow()
		return w.Start(input)
	}
	openID4VCIIssuerWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewOpenID4VCIIssuerWorkflow()
		return w.Start(input)
	}
	openID4VPVerifierWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewOpenID4VPVerifierWorkflow()
		return w.Start(input)
	}
	ewcWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewEWCWorkflow()
		return w.Start(input)
	}
	webuildWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewWebuildWorkflow()
		return w.Start(input)
	}
	eudiwWorkflowStart = func(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewEudiwWorkflow()
		return w.Start(input)
	}
	vleiWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewVLEIValidationWorkflow()
		return w.Start(namespace, input)
	}
	customCheckWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewCustomCheckWorkflow()
		return w.Start(namespace, input)
	}
)

func HandleSaveVariablesAndStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[SaveVariablesAndStartRequestInput](e)
		if err != nil {
			return err
		}

		userID := e.Auth.Id
		email := e.Auth.GetString("email")
		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			)
		}
		orgID, err := pbutils.GetUserOrganizationID(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization ID",
				err.Error(),
			)
		}
		userName := e.Auth.GetString("name")

		protocol := e.Request.PathValue("protocol")
		version := e.Request.PathValue("version")
		if protocol == "" || version == "" {
			return apierror.New(
				http.StatusBadRequest,
				"protocol and version",
				"protocol and version are required",
				"missing parameters",
			)
		}

		if !filepath.IsLocal(filepath.Join(protocol, version)) {
			return apierror.New(
				http.StatusBadRequest,
				"protocol and version",
				"invalid protocol or version",
				"invalid protocol or version",
			)
		}
		rootDir := utils.GetEnvironmentVariable("ROOT_DIR", ".")
		dirPath := filepath.Join(rootDir, "config_templates", protocol, version)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return apierror.New(
				http.StatusBadRequest,
				"directory",
				"directory does not exist for test "+protocol+"/"+version,
				"directory does not exist",
			)
		}

		var returns []workflowengine.WorkflowResult

		for testName, config := range req.ConfigsWithJSON {
			author := Author(strings.Split(testName, "/")[0])
			if author == "" {
				return apierror.New(
					http.StatusBadRequest,
					"author",
					"author is required",
					"missing author",
				)
			}
			memo := map[string]interface{}{
				"test":     testName,
				"standard": protocol,
				"author":   author,
			}

			results, err := processJSONChecks(
				e.App,
				config,
				email,
				namespace,
				memo,
				author,
				testName,
				protocol,
				version,
				userName,
			)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"json",
					"failed to process JSON checks",
					err.Error(),
				)
			}
			returns = append(returns, results)
		}

		for testName, testData := range req.ConfigsWithFields {
			author := Author(strings.Split(testName, "/")[0])
			if author == "" {
				return apierror.New(
					http.StatusBadRequest,
					"author",
					"author is required",
					"missing author",
				)
			}
			memo := map[string]interface{}{
				"test":     testName,
				"standard": protocol,
				"author":   author,
			}
			results, err := processVariablesTest(
				e.App,
				testName,
				testData,
				email,
				namespace,
				dirPath,
				memo,
				author,
				protocol,
				version,
				userName,
				orgID,
			)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"variables",
					"failed to process variables test",
					err.Error(),
				)
			}
			returns = append(returns, results)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"protocol/version": protocol + "/" + version,
			"message":          "Tests started successfully",
			"results":          returns,
		})
	}
}

func reduceData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if strValue, ok := value.(string); ok {
				trimmed := strings.TrimSpace(strValue)
				var jsonValue interface{}
				if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
					(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
					if err := json.Unmarshal([]byte(trimmed), &jsonValue); err == nil {
						v[key] = reduceData(jsonValue)
						continue
					}
				}
				v[key] = trimmed
			} else {
				v[key] = reduceData(value)
			}
		}
		return v

	case []interface{}:
		for i, value := range v {
			if strValue, ok := value.(string); ok {
				trimmed := strings.TrimSpace(strValue)
				var jsonValue interface{}
				if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
					(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
					if err := json.Unmarshal([]byte(trimmed), &jsonValue); err == nil {
						v[i] = reduceData(jsonValue)
						continue
					}
				}
				v[i] = trimmed
			} else {
				v[i] = reduceData(value)
			}
		}
		return v

	default:
		return data
	}
}

func startOpenID4VPWalletWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	// Automated OpenID conformance checks share the openid_conformance_suite author key
	// but use a dedicated workflow and input format.
	if i.Protocol == "openid4vci_issuer" {
		return startOpenIDAutomatedConformanceWorkflow(
			i,
			workflows.OpenID4VCIIssuerStepCITemplatePath,
			openID4VCIIssuerWorkflowStart,
			"OID4VCI issuer",
		)
	}
	if i.Protocol == "openid4vp_verifier" {
		return startOpenIDAutomatedConformanceWorkflow(
			i,
			workflows.OpenID4VPVerifierStepCITemplatePath,
			openID4VPVerifierWorkflowStart,
			"OID4VP verifier",
		)
	}

	yamlData := i.YAMLData
	email := i.Email
	namespace := i.Namespace
	memo := i.Memo
	version := i.Version

	if yamlData == "" {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"YAML data is required for vLEI workflow",
			"missing YAML data",
		)
	}
	var data interface{}

	err := yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse YAML input",
			err.Error(),
		)
	}

	dataMap := reduceData(data)

	jsonDataFinal, err := json.Marshal(dataMap)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to convert YAML to JSON",
			err.Error(),
		)
	}

	var parsedData openID4VPTestInputFile
	if err := json.Unmarshal(jsonDataFinal, &parsedData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to parse JSON input",
			err.Error(),
		)
	}

	var templateStr string
	switch version {
	case "1.0":
		templateStr, err = readTemplateFile(
			utils.GetEnvironmentVariable("ROOT_DIR", "."),
			workflows.OpenID4VPWalletStepCITemplatePathv1_0,
		)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
	case "draft-24":
		templateStr, err = readTemplateFile(
			utils.GetEnvironmentVariable("ROOT_DIR", "."),
			workflows.OpenID4VPWalletStepCITemplatePathDr24,
		)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
	default:
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"version",
			"invalid version",
			"invalid version",
		)
	}

	input := workflowengine.WorkflowInput{
		Payload: workflows.OpenID4VPWalletWorkflowPayload{
			Variant:  string(parsedData.Variant),
			Form:     parsedData.Form,
			TestName: parsedData.TestName,
			UserMail: email,
		},
		Config: workflowengine.WithAppConfig(i.App, map[string]any{
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
			"user_name": i.UserName,
		}),
	}
	results, err := openID4VPWalletWorkflowStart(input)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}
	results.Author = string(i.Author)
	return results, nil
}

func startOpenIDAutomatedConformanceWorkflow(
	i WorkflowStarterParams,
	templatePath string,
	starter func(workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error),
	label string,
) (workflowengine.WorkflowResult, error) {
	if i.YAMLData == "" {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			fmt.Sprintf("YAML data is required for %s workflow", label),
			"missing YAML data",
		)
	}

	templateStr, err := readTemplateFile(
		utils.GetEnvironmentVariable("ROOT_DIR", "."),
		templatePath,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	var rawData map[string]any
	if err := yaml.Unmarshal([]byte(i.YAMLData), &rawData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			fmt.Sprintf("failed to parse YAML input for %s workflow", label),
			err.Error(),
		)
	}

	input := workflowengine.WorkflowInput{
		Payload: workflows.OpenIDConformanceWorkflowPayload{
			Parameters: conformanceInputParameters(rawData, map[string]struct{}{"test": {}}),
			TestName:   conformanceInputString(rawData, "test"),
			UserMail:   i.Email,
		},
		Config: workflowengine.WithAppConfig(i.App, map[string]any{
			"template":  templateStr,
			"namespace": i.Namespace,
			"memo":      i.Memo,
			"user_name": i.UserName,
		}),
	}

	results, err := starter(input)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			fmt.Sprintf("failed to start %s workflow", label),
			err.Error(),
		)
	}
	results.Author = string(i.Author)
	return results, nil
}

func startEWCWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	return startEWCLikeWorkflow(
		i,
		workflows.EWCSuite,
		workflows.EWCTemplateFolderPath,
		ewcWorkflowStart,
	)
}

func startWebuildWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	return startEWCLikeWorkflow(
		i,
		workflows.WebuildSuite,
		workflows.WebuildTemplateFolderPath,
		webuildWorkflowStart,
	)
}

func startEWCLikeWorkflow(
	i WorkflowStarterParams,
	suite string,
	templateFolderPath string,
	starter func(workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error),
) (workflowengine.WorkflowResult, error) {
	filename := strings.TrimPrefix(
		strings.TrimSuffix(i.TestName, filepath.Ext(i.TestName))+".yaml",
		suite,
	)
	filename = strings.TrimPrefix(filename, "/")
	templateStr, err := readTemplateFile(
		filepath.Join(utils.GetEnvironmentVariable("ROOT_DIR", "."), templateFolderPath),
		filename,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	var rawData map[string]any
	if err := yaml.Unmarshal([]byte(i.YAMLData), &rawData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse YAML input",
			err.Error(),
		)
	}

	checkEndpoint, err := workflows.ResolveEWCLikeCheckEndpoint(suite, i.Protocol)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"protocol",
			err.Error(),
			"unsupported protocol",
		)
	}
	logsEndpoint, err := workflows.ResolveEWCLikeLogsEndpoint(suite, i.Protocol)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"protocol",
			err.Error(),
			"unsupported protocol",
		)
	}

	input := workflowengine.WorkflowInput{
		Payload: workflows.EWCWorkflowPayload{
			Parameters: conformanceInputParameters(rawData, nil),
			UserMail:   i.Email,
		},
		Config: workflowengine.WithAppConfig(i.App, map[string]any{
			"template":       templateStr,
			"namespace":      i.Namespace,
			"memo":           i.Memo,
			"version":        i.Version,
			"check_endpoint": checkEndpoint,
			"logs_endpoint":  logsEndpoint,
			"user_name":      i.UserName,
		}),
	}
	results, err := starter(input)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}
	results.Author = string(i.Author)
	return results, nil
}

func startEudiwWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	yamlData := i.YAMLData
	email := i.Email
	namespace := i.Namespace
	memo := i.Memo
	testName := i.TestName
	filename := strings.TrimPrefix(
		strings.TrimSuffix(testName, filepath.Ext(testName))+".yaml",
		workflows.EudiwSuite,
	)
	filename = strings.TrimPrefix(filename, "/")
	templateStr, err := readTemplateFile(
		filepath.Join(
			utils.GetEnvironmentVariable("ROOT_DIR", "."),
			workflows.EudiwTemplateFolderPath,
		),
		filename,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	var parsedData EudiwInput
	if err := yaml.Unmarshal([]byte(yamlData), &parsedData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse YAML input",
			err.Error(),
		)
	}
	input := workflowengine.WorkflowInput{
		Payload: workflows.EudiwWorkflowPayload{
			Nonce:    parsedData.Nonce,
			ID:       parsedData.ID,
			UserMail: email,
		},
		Config: workflowengine.WithAppConfig(i.App, map[string]any{
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
			"user_name": i.UserName,
		}),
	}
	results, err := eudiwWorkflowStart(input)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}
	results.Author = string(i.Author)
	return results, nil
}
func startvLEIWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	yamlData := i.YAMLData
	namespace := i.Namespace
	memo := i.Memo

	if yamlData == "" {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"YAML data is required for OID4VP wallet workflow",
			"missing YAML data",
		)
	}
	var data interface{}

	err := yaml.Unmarshal([]byte(yamlData), &data)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse YAML input",
			err.Error(),
		)
	}

	dataMap := reduceData(data)

	jsonDataFinal, err := json.Marshal(dataMap)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to convert YAML to JSON",
			err.Error(),
		)
	}

	var parsedData vLEICheckInput
	if err := json.Unmarshal(jsonDataFinal, &parsedData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to parse JSON input",
			err.Error(),
		)
	}
	input := workflowengine.WorkflowInput{
		Config: workflowengine.WithAppConfig(i.App, map[string]any{
			"server_url": parsedData.ServerURL,
			"memo":       memo,
		}),
		Payload: workflows.VLEIValidationWorkflowPayload{
			CredentialID: parsedData.CredentialID,
		},
	}

	results, err := vleiWorkflowStart(namespace, input)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}
	results.Author = string(i.Author)
	return results, nil
}

func processJSONChecks(
	app core.App,
	testData string,
	email string,
	namespace string,
	memo map[string]interface{},
	author Author,
	testName string,
	protocol string,
	version string,
	userName string,
) (workflowengine.WorkflowResult, error) {
	input := WorkflowStarterParams{
		App:       app,
		YAMLData:  testData,
		Email:     email,
		Namespace: namespace,
		Memo:      memo,
		Author:    author,
		TestName:  testName,
		Protocol:  protocol,
		Version:   version,
		UserName:  userName,
	}
	if starterFunc, ok := workflowRegistry[author]; ok {
		return starterFunc(input)
	}
	return workflowengine.WorkflowResult{}, apierror.New(
		http.StatusBadRequest,
		"author",
		"unsupported author for test "+testName,
		"unsupported author",
	)
}

func processVariablesTest(
	app core.App,
	testName string,
	variables []Variable,
	email string,
	namespace string,
	dirPath string,
	memo map[string]interface{},
	author Author,
	protocol string,
	version string,
	userName string,
	orgID string,
) (workflowengine.WorkflowResult, error) {
	templateData, err := readFileInDir(dirPath, testName)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"template",
			"failed to open template for test "+testName,
			"failed to open template",
		)
	}

	values := make(map[string]interface{})
	configValues, err := app.FindCollectionByNameOrId("config_values")
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	for _, variable := range variables {
		record := core.NewRecord(configValues)
		record.Set("credimi_id", variable.CredimiID)
		record.Set("value", variable.Value)
		record.Set("field_name", variable.FieldName)
		record.Set("template_path", testName)
		record.Set("owner", orgID)
		if err := app.Save(record); err != nil {
			return workflowengine.WorkflowResult{}, apierror.New(
				http.StatusBadRequest,
				"save",
				"failed to save variable for test "+testName,
				err.Error(),
			)
		}
		values[variable.FieldName] = variable.Value
	}

	for key, value := range values {
		if strValue, ok := value.(string); ok {
			values[key] = strings.ReplaceAll(strValue, "\n", "")
		}
	}

	renderedTemplate, err := engine.RenderTemplate(bytes.NewReader(templateData), values)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"template",
			"failed to render template for test "+testName,
			err.Error(),
		)
	}

	input := WorkflowStarterParams{
		App:       app,
		YAMLData:  renderedTemplate,
		Email:     email,
		Namespace: namespace,
		Memo:      memo,
		Author:    author,
		TestName:  testName,
		Protocol:  protocol,
		Version:   version,
		UserName:  userName,
	}

	if starterFunc, ok := workflowRegistry[author]; ok {
		return starterFunc(input)
	}
	return workflowengine.WorkflowResult{}, apierror.New(
		http.StatusBadRequest,
		"author",
		"unsupported author for test "+testName,
		"unsupported author",
	)
}

// readTemplateFile reads the template name inside baseDir. Names that would
// resolve outside baseDir are rejected, and the error never exposes server paths.
func readTemplateFile(baseDir, name string) (string, error) {
	data, err := readFileInDir(baseDir, name)
	if err != nil {
		return "", apierror.New(
			http.StatusBadRequest,
			"file",
			"failed to read template file",
			"failed to read template file",
		)
	}
	return string(data), nil
}

// readFileInDir reads name relative to baseDir. Absolute names, ".." escapes and
// symlinks leading outside baseDir are rejected.
func readFileInDir(baseDir, name string) ([]byte, error) {
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("template name is not local: %q", name)
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, fmt.Errorf("open template dir: %w", err)
	}
	defer root.Close()
	data, err := root.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	return data, nil
}
