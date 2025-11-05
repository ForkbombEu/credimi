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
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	engine "github.com/forkbombeu/credimi/pkg/templateengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

type CustomCheck struct {
	Form interface{} `json:"form"`
	Yaml string      `json:"yaml" validate:"required"`
}
type Variable struct {
	FieldName string      `json:"field_name" validate:"required"`
	Value     interface{} `json:"value"      validate:"required"`
	CredimiID string      `json:"credimi_id" validate:"required"`
}

type SaveVariablesAndStartRequestInput struct {
	ConfigsWithFields map[string][]Variable  `json:"configs_with_fields" validate:"required"`
	ConfigsWithJSON   map[string]string      `json:"configs_with_json"   validate:"required"`
	CustomChecks      map[string]CustomCheck `json:"custom_checks"       validate:"required"`
}

type openID4VPTestInputFile struct {
	Variant  json.RawMessage `json:"variant" yaml:"variant" validate:"required,oneof=json variables yaml"`
	Form     any             `json:"form"    yaml:"form"`
	TestName string          `json:"test" yaml:"test" validate:"required"`
}
type vLEICheckInput struct {
	CredentialID string `json:"credentialID"`
	ServerURL    string `json:"serverURL"`
}

type EWCInput struct {
	SessionID string `yaml:"sessionId" json:"sessionId" validate:"required"`
}

type EudiwInput struct {
	Nonce string `json:"nonce" yaml:"nonce" validate:"required"`
	ID    string `json:"id"    yaml:"id"    validate:"required"`
}

type Author string

type WorkflowStarterParams struct {
	YAMLData  string
	Email     string
	AppURL    string
	Namespace string
	Memo      map[string]interface{}
	Author    Author
	TestName  string
	Protocol  string
	Version   string
	AppName   string
	LogoUrl   string
	UserName  string
}

type WorkflowStarter func(params WorkflowStarterParams) (workflowengine.WorkflowResult, error)

var workflowRegistry = map[Author]WorkflowStarter{
	"ewc":                      startEWCWorkflow,
	"openid_conformance_suite": startOpenIDNetWorkflow,
	"eudiw":                    startEudiwWorkflow,
	"vlei":                     startvLEIWorkflow,
}

func HandleSaveVariablesAndStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[SaveVariablesAndStartRequestInput](e)
		if err != nil {
			return err
		}

		appURL := e.App.Settings().Meta.AppURL
		userID := e.Auth.Id
		email := e.Auth.GetString("email")
		namespace, err := GetUserOrganizationCanonifiedName(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		orgID, err := GetUserOrganizationID(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization ID",
				err.Error(),
			).JSON(e)
		}
		appName := e.App.Settings().Meta.AppName
		logoUrl := fmt.Sprintf(
			"%s/logos/%s_logo-transp_emblem.png",
			appURL,
			strings.ToLower(appName),
		)
		userName := e.Auth.GetString("name")

		protocol := e.Request.PathValue("protocol")
		version := e.Request.PathValue("version")
		if protocol == "" || version == "" {
			return apierror.New(
				http.StatusBadRequest,
				"protocol and version",
				"protocol and version are required",
				"missing parameters",
			).JSON(e)
		}

		dirPath := os.Getenv("ROOT_DIR") + "/config_templates/" + protocol + "/" + version + "/"
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return apierror.New(
				http.StatusBadRequest,
				"directory",
				"directory does not exist for test "+os.Getenv("ROOT_DIR")+protocol+"/"+version,
				err.Error(),
			).JSON(e)
		}

		var returns []workflowengine.WorkflowResult

		for id, customCheck := range req.CustomChecks {
			if customCheck.Yaml == "" {
				return apierror.New(
					http.StatusBadRequest,
					"yaml",
					"yaml is required for custom check",
					"missing yaml",
				).JSON(e)
			}
			var formJSON string
			if customCheck.Form != nil {
				b, err := json.Marshal(customCheck.Form)
				if err != nil {
					return apierror.New(
						http.StatusBadRequest,
						"form",
						"failed to serialize form to JSON",
						err.Error(),
					).JSON(e)
				}
				formJSON = string(b)
			}

			memo := map[string]interface{}{
				"test":     "custom-check",
				"standard": protocol,
				"author":   id,
				"version":  version,
			}
			results, err := processCustomChecks(
				customCheck.Yaml,
				appURL,
				namespace,
				memo,
				formJSON,
			)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"custom-check",
					"failed to process custom check",
					err.Error(),
				).JSON(e)
			}
			returns = append(returns, results)
		}

		for testName, config := range req.ConfigsWithJSON {
			author := Author(strings.Split(testName, "/")[0])
			if author == "" {
				return apierror.New(
					http.StatusBadRequest,
					"author",
					"author is required",
					"missing author",
				).JSON(e)
			}
			memo := map[string]interface{}{
				"test":     testName,
				"standard": protocol,
				"author":   author,
			}

			results, err := processJSONChecks(
				config,
				email,
				appURL,
				namespace,
				memo,
				author,
				testName,
				protocol,
				version,
				logoUrl,
				appName,
				userName,
			)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"json",
					"failed to process JSON checks",
					err.Error(),
				).JSON(e)
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
				).JSON(e)
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
				appURL,
				namespace,
				dirPath,
				memo,
				author,
				protocol,
				version,
				logoUrl,
				appName,
				userName,
				orgID,
			)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"variables",
					"failed to process variables test",
					err.Error(),
				).JSON(e)
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

func startOpenIDNetWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	yamlData := i.YAMLData
	email := i.Email
	appURL := i.AppURL
	namespace := i.Namespace
	memo := i.Memo
	version := i.Version

	if yamlData == "" {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"YAML data is required for OpenIDNet workflow",
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
			os.Getenv("ROOT_DIR") + "/" + workflows.OpenIDNetStepCITemplatePathv1_0,
		)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
	case "draft-24":
		templateStr, err = readTemplateFile(
			os.Getenv("ROOT_DIR") + "/" + workflows.OpenIDNetStepCITemplatePathDr24,
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
		Payload: map[string]any{
			"variant":   string(parsedData.Variant),
			"form":      parsedData.Form,
			"test_name": parsedData.TestName,
			"user_mail": email,
		},
		Config: map[string]any{
			"app_url":   appURL,
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
			"app_name":  i.AppName,
			"app_logo":  i.LogoUrl,
			"user_name": i.UserName,
		},
	}
	var workflow workflows.OpenIDNetWorkflow
	results, err := workflow.Start(input)
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

func startEWCWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	yamlData := i.YAMLData
	email := i.Email
	appURL := i.AppURL
	namespace := i.Namespace
	memo := i.Memo
	protocol := i.Protocol
	testName := i.TestName
	filename := strings.TrimPrefix(
		strings.TrimSuffix(testName, filepath.Ext(testName))+".yaml",
		"ewc",
	)
	templateStr, err := readTemplateFile(
		os.Getenv("ROOT_DIR") + "/" + workflows.EWCTemplateFolderPath + filename,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	var parsedData EWCInput
	if err := yaml.Unmarshal([]byte(yamlData), &parsedData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse YAML input",
			err.Error(),
		)
	}
	var checkEndpoint string
	switch protocol {
	case "openid4vp_wallet":
		checkEndpoint = "https://ewc.api.forkbomb.eu/verificationStatus"
	case "openid4vci_wallet":
		checkEndpoint = "https://ewc.api.forkbomb.eu/issueStatus"
	default:
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"protocol",
			fmt.Sprintf("unsupported protocol %s for EWC suite", protocol),
			"unsupported protocol",
		)
	}
	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"session_id": parsedData.SessionID,
			"user_mail":  email,
		},
		Config: map[string]any{
			"app_url":        appURL,
			"template":       templateStr,
			"namespace":      namespace,
			"memo":           memo,
			"check_endpoint": checkEndpoint,
			"app_name":       i.AppName,
			"app_logo":       i.LogoUrl,
			"user_name":      i.UserName,
		},
	}
	var workflow workflows.EWCWorkflow
	results, err := workflow.Start(input)
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
	appURL := i.AppURL
	namespace := i.Namespace
	memo := i.Memo
	testName := i.TestName
	filename := strings.TrimPrefix(
		strings.TrimSuffix(testName, filepath.Ext(testName))+".yaml",
		"eudiw",
	)
	templateStr, err := readTemplateFile(
		os.Getenv("ROOT_DIR") + "/" + workflows.EudiwTemplateFolderPath + filename,
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
		Payload: map[string]any{
			"nonce":     parsedData.Nonce,
			"id":        parsedData.ID,
			"user_mail": email,
		},
		Config: map[string]any{
			"app_url":   appURL,
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
			"app_name":  i.AppName,
			"app_logo":  i.LogoUrl,
			"user_name": i.UserName,
		},
	}
	var workflow workflows.EudiwWorkflow
	results, err := workflow.Start(input)
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
	appURL := i.AppURL
	namespace := i.Namespace
	memo := i.Memo

	if yamlData == "" {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"YAML data is required for OpenIDNet workflow",
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
		Config: map[string]any{
			"app_url":    appURL,
			"server_url": parsedData.ServerURL, // TODO update with the real one
			"memo":       memo,
		},
		Payload: map[string]any{
			"credentialID": parsedData.CredentialID,
		},
	}

	var workflow workflows.VLEIValidationWorkflow
	results, err := workflow.Start(namespace, input)
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
	testData string,
	email,
	appURL string,
	namespace string,
	memo map[string]interface{},
	author Author,
	testName string,
	protocol string,
	version string,
	logoUrl string,
	appName string,
	userName string,
) (workflowengine.WorkflowResult, error) {
	input := WorkflowStarterParams{
		YAMLData:  testData,
		Email:     email,
		AppURL:    appURL,
		Namespace: namespace,
		Memo:      memo,
		Author:    author,
		TestName:  testName,
		Protocol:  protocol,
		Version:   version,
		LogoUrl:   logoUrl,
		AppName:   appName,
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
	email,
	appURL string,
	namespace string,
	dirPath string,
	memo map[string]interface{},
	author Author,
	protocol string,
	version string,
	logoUrl string,
	appName string,
	userName string,
	orgID string,
) (workflowengine.WorkflowResult, error) {
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

	templatePath := dirPath + testName
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"template",
			"failed to open template for test "+testName,
			err.Error(),
		)
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
		YAMLData:  renderedTemplate,
		Email:     email,
		AppURL:    appURL,
		Namespace: namespace,
		Memo:      memo,
		Author:    author,
		TestName:  testName,
		Protocol:  protocol,
		Version:   version,
		LogoUrl:   logoUrl,
		AppName:   appName,
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

func processCustomChecks(
	testData string,
	appURL string,
	namespace string,
	memo map[string]interface{},
	formJSON string,
) (workflowengine.WorkflowResult, error) {
	yaml := testData
	if yaml == "" {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"yaml is empty",
			"yaml is empty",
		)
	}

	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"yaml": yaml,
		},
		Config: map[string]any{
			"memo":    memo,
			"app_url": appURL,
			"env":     formJSON,
		},
	}

	var w workflows.CustomCheckWorkflow

	results, errStart := w.Start(namespace, input)
	if errStart != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start check",
			errStart.Error(),
		)
	}
	authorVal, ok := memo["author"]
	if ok {
		results.Author = fmt.Sprintf("%v", authorVal)
	}
	return results, nil
}

func readTemplateFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", apierror.New(
			http.StatusBadRequest,
			"file",
			"failed to read template file",
			err.Error(),
		)
	}
	return string(data), nil
}
