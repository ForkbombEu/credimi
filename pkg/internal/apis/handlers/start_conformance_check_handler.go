// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
)

type CustomCheck struct {
	Form interface{} `json:"form" validate:"required"`
	Yaml string      `json:"yaml" validate:"required"`
}
type Variable struct {
	FieldName string      `json:"field_name" validate:"required"`
	Value     interface{} `json:"value" validate:"required"`
	CredimiID string      `json:"credimi_id" validate:"required"`
}

type SaveVariablesAndStartRequestInput struct {
	ConfigsWithFields map[string][]Variable  `json:"configs_with_fields" validate:"required"`
	ConfigsWithJSON   map[string]string      `json:"configs_with_json" validate:"required"`
	CustomChecks      map[string]CustomCheck `json:"custom_checks" validate:"required"`
}

type openID4VPTestInputFile struct {
	Variant json.RawMessage `json:"variant" validate:"required,oneof=json variables yaml"`
	Form    any             `json:"form"`
}

type EWCInput struct {
	SessionID string `json:"sessionId" validate:"required"`
}

type EudiwInput struct {
	Nonce string `json:"nonce" validate:"required"`
	ID    string `json:"id"    validate:"required"`
}

type Author string

type WorkflowStarterParams struct {
	JSONData  string
	Email     string
	AppURL    string
	Namespace interface{}
	Memo      map[string]interface{}
	Author    Author
	TestName  string
	Protocol  string
}

type WorkflowStarter func(params WorkflowStarterParams) (workflowengine.WorkflowResult, error)

var workflowRegistry = map[Author]WorkflowStarter{
	"ewc":                      startEWCWorkflow,
	"openid_conformance_suite": startOpenIDNetWorkflow,
	"eudiw":                    startEudiwWorkflow,
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
		namespace, err := GetUserOrganizationID(e.App, userID)
		if err != nil {
			return err
		}

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

		dirPath := os.Getenv("ROOT_DIR") + "/config_templates/" + protocol + "/" + version + "/"
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return apierror.New(
				http.StatusBadRequest,
				"directory",
				"directory does not exist for test "+os.Getenv("ROOT_DIR")+protocol+"/"+version,
				err.Error(),
			)
		}

		var returns []workflowengine.WorkflowResult

		for id, customCheck := range req.CustomChecks {
			if customCheck.Yaml == "" {
				return apierror.New(
					http.StatusBadRequest,
					"yaml",
					"yaml is required for custom check",
					"missing yaml",
				)
			}
			if customCheck.Form == nil {
				return apierror.New(
					http.StatusBadRequest,
					"form",
					"form is required for custom check",
					"missing form",
				)
			}
			log.Println("Custom check form:", customCheck.Form)
			formJSON, err := json.Marshal(customCheck.Form)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"form",
					"failed to serialize form to JSON",
					err.Error(),
				)
			}

			memo := map[string]interface{}{
				"test":     "custom-check",
				"standard": protocol,
				"author":   id,
			}
			results, err := processCustomChecks(
				customCheck.Yaml,
				appURL,
				namespace,
				memo,
				string(formJSON),
			)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"custom-check",
					"failed to process custom check",
					err.Error(),
				)
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
				)
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
				appURL,
				namespace,
				dirPath,
				memo,
				author,
				protocol,
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

func startOpenIDNetWorkflow(i WorkflowStarterParams) (workflowengine.WorkflowResult, error) {
	jsonData := i.JSONData
	email := i.Email
	appURL := i.AppURL
	namespace := i.Namespace
	memo := i.Memo

	var parsedData openID4VPTestInputFile
	if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to parse JSON input",
			err.Error(),
		)
	}
	templateStr, err := readTemplateFile(
		os.Getenv("ROOT_DIR") + "/" + workflows.OpenIDNetStepCITemplatePath,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"variant":   string(parsedData.Variant),
			"form":      parsedData.Form,
			"user_mail": email,
			"app_url":   appURL,
		},
		Config: map[string]any{
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
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
	jsonData := i.JSONData
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
	if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to parse JSON input",
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
			"app_url":    appURL,
		},
		Config: map[string]any{
			"template":       templateStr,
			"namespace":      namespace,
			"memo":           memo,
			"check_endpoint": checkEndpoint,
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
	jsonData := i.JSONData
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
	if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
		return workflowengine.WorkflowResult{}, apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to parse JSON input",
			err.Error(),
		)
	}
	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"nonce":     parsedData.Nonce,
			"id":        parsedData.ID,
			"user_mail": email,
			"app_url":   appURL,
		},
		Config: map[string]any{
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
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

func processJSONChecks(
	testData string,
	email,
	appURL string,
	namespace interface{},
	memo map[string]interface{},
	author Author,
	testName string,
	protocol string,
) (workflowengine.WorkflowResult, error) {
	input := WorkflowStarterParams{
		JSONData:  testData,
		Email:     email,
		AppURL:    appURL,
		Namespace: namespace,
		Memo:      memo,
		Author:    author,
		TestName:  testName,
		Protocol:  protocol,
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
	namespace interface{},
	dirPath string,
	memo map[string]interface{},
	author Author,
	protocol string,
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
		record.Set("owner", namespace)
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
		JSONData:  renderedTemplate,
		Email:     email,
		AppURL:    appURL,
		Namespace: namespace,
		Memo:      memo,
		Author:    author,
		TestName:  testName,
		Protocol:  protocol,
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
	namespace interface{},
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
	log.Println("Custom check form JSON:", formJSON)

	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"yaml": yaml,
		},
		Config: map[string]any{
			"namespace": namespace,
			"memo":      memo,
			"app_url":   appURL,
			"env":       formJSON,
		},
	}

	var w workflows.CustomCheckWorkflow

	results, errStart := w.Start(input)
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
