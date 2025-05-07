// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"encoding/json"
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

type SaveVariablesAndStartRequestInput map[string]struct {
	Format string      `json:"format" validate:"required"`
	Data   interface{} `json:"data" validate:"required"`
}

type openID4VPTestInputFile struct {
	Variant json.RawMessage `json:"variant"  validate:"required,oneof=json variables yaml"`
	Form    any             `json:"form"`
}

type EWCInput struct {
	SessionID string `json:"sessionId" validate:"required"`
}

type Author string

type WorkflowStarterParams struct {
	JsonData  string
	Email     string
	AppURL    string
	Namespace interface{}
	Memo      map[string]interface{}
	Author    Author
	TestName  string
	Protocol  string
}

type WorkflowStarter func(params WorkflowStarterParams) error

var workflowRegistry = map[Author]WorkflowStarter{
	"ewc":                      startEWCWorkflow,
	"openid_conformance_suite": startOpenIDNetWorkflow,
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
		namespace, err := getUserNamespace(e.App, userID)
		if err != nil {
			return err
		}

		protocol := e.Request.PathValue("protocol")
		author := e.Request.PathValue("author")
		if protocol == "" || author == "" {
			return apierror.New(
				http.StatusBadRequest,
				"protocol and author",
				"protocol and author are required",
				"missing parameters",
			)
		}

		dirPath := os.Getenv("ROOT_DIR") + "/config_templates/" + protocol + "/" + author + "/"
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return apierror.New(
				http.StatusBadRequest,
				"directory",
				"directory does not exist for test "+os.Getenv("ROOT_DIR")+protocol+"/"+author,
				err.Error(),
			)
		}

		for testName, testData := range req {
			memo := map[string]interface{}{
				"test":     testName,
				"standard": protocol,
				"author":   author,
			}
			suite := Author(strings.Split(testName, "/")[0])
			if suite == "" {
				return apierror.New(
					http.StatusBadRequest,
					"suite",
					"suite is required",
					"missing suite",
				)
			}

			switch testData.Format {
			case "json":
				if err := processJSONChecks(testData, email, appURL, namespace, memo, suite, testName, protocol); err != nil {
					return apierror.New(
						http.StatusBadRequest,
						"json",
						"failed to process JSON checks",
						err.Error(),
					)
				}
			case "variables":
				if err := processVariablesTest(
					e.App,
					testName,
					testData,
					email,
					appURL,
					namespace,
					dirPath,
					memo,
					suite,
					protocol,
				); err != nil {
					return apierror.New(
						http.StatusBadRequest,
						"variables",
						"failed to process variables test",
						err.Error(),
					)
				}
			default:
				return apierror.New(
					http.StatusBadRequest,
					"format",
					"unsupported format for test "+testName,
					"unsupported format",
				)
			}
		}

		return e.JSON(http.StatusOK, map[string]bool{"started": true})
	}
}

func startOpenIDNetWorkflow(i WorkflowStarterParams) error {
	jsonData := i.JsonData
	email := i.Email
	appURL := i.AppURL
	namespace := i.Namespace
	memo := i.Memo

	var parsedData openID4VPTestInputFile
	if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
		return apierror.New(
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
		return err
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
	if _, err := workflow.Start(input); err != nil {
		return apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}
	return nil
}

func startEWCWorkflow(i WorkflowStarterParams) error {
	jsonData := i.JsonData
	email := i.Email
	appURL := i.AppURL
	namespace := i.Namespace
	memo := i.Memo
	protocol := i.Protocol
	testName := i.TestName

	filename := strings.TrimPrefix(strings.TrimSuffix(testName, filepath.Ext(testName))+".yaml", "ewc")
	templateStr, err := readTemplateFile(
		os.Getenv("ROOT_DIR") + "/pkg/workflowengine/workflows/ewc_config" + filename,
	)
	if err != nil {
		return err
	}
	var parsedData EWCInput
	if err := json.Unmarshal([]byte(jsonData), &parsedData); err != nil {
		return apierror.New(
			http.StatusBadRequest,
			"json",
			"failed to parse JSON input",
			err.Error(),
		)
	}

	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"session_id": parsedData.SessionID,
			"user_mail":  email,
			"app_url":    appURL,
		},
		Config: map[string]any{
			"template":  templateStr,
			"namespace": namespace,
			"memo":      memo,
			"protocol":  protocol,
		},
	}
	var workflow workflows.EWCWorkflow
	if _, err := workflow.Start(input); err != nil {
		return apierror.New(
			http.StatusBadRequest,
			"workflow",
			"failed to start workflow",
			err.Error(),
		)
	}
	return nil
}

func processJSONChecks(
	testData struct {
		Format string      `json:"format" validate:"required"`
		Data   interface{} `json:"data" validate:"required"`
	},
	email,
	appURL string,
	namespace interface{},
	memo map[string]interface{},
	author Author,
	testName string,
	protocol string,
) error {
	jsonData, ok := testData.Data.(string)
	if !ok {
		return apierror.New(
			http.StatusBadRequest,
			"json",
			"invalid JSON format for test "+testName,
			"invalid JSON format",
		)
	}
	input := WorkflowStarterParams{
		JsonData:  jsonData,
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
	return apierror.New(
		http.StatusBadRequest,
		"author",
		"unsupported author for test "+testName,
		"unsupported author",
	)
}

func processVariablesTest(
	app core.App,
	testName string,
	testData struct {
		Format string      `json:"format" validate:"required"`
		Data   interface{} `json:"data" validate:"required"`
	},
	email,
	appURL string,
	namespace interface{},
	dirPath string,
	memo map[string]interface{},
	author Author,
	protocol string,
) error {
	variables, ok := testData.Data.(map[string]interface{})
	if !ok {
		return apierror.New(
			http.StatusBadRequest,
			"variables",
			"invalid variables format for test "+testName,
			"invalid variables format",
		)
	}

	values := make(map[string]interface{})
	configValues, err := app.FindCollectionByNameOrId("config_values")
	if err != nil {
		return err
	}

	for credimiID, variable := range variables {
		v, ok := variable.(map[string]interface{})
		if !ok {
			return apierror.New(
				http.StatusBadRequest,
				"variable",
				"invalid variable format for test "+testName,
				"invalid variable format",
			)
		}
		fieldName, ok := v["fieldName"].(string)
		if !ok {
			return apierror.New(
				http.StatusBadRequest,
				"fieldName",
				"invalid fieldName format for test "+testName,
				"invalid fieldName format",
			)
		}

		record := core.NewRecord(configValues)
		record.Set("credimi_id", credimiID)
		record.Set("value", v["value"])
		record.Set("field_name", fieldName)
		record.Set("template_path", testName)
		if err := app.Save(record); err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"save",
				"failed to save variable for test "+testName,
				err.Error(),
			)
		}
		values[fieldName] = v["value"]
	}

	templatePath := dirPath + testName
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return apierror.New(
			http.StatusBadRequest,
			"template",
			"failed to open template for test "+testName,
			err.Error(),
		)
	}

	renderedTemplate, err := engine.RenderTemplate(bytes.NewReader(templateData), values)
	if err != nil {
		return apierror.New(
			http.StatusBadRequest,
			"template",
			"failed to render template for test "+testName,
			err.Error(),
		)
	}

	input := WorkflowStarterParams{
		JsonData:  renderedTemplate,
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
	return apierror.New(
		http.StatusBadRequest,
		"author",
		"unsupported author for test "+testName,
		"unsupported author",
	)
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
