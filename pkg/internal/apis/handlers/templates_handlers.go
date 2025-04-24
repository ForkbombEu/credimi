// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later


package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	p "path"


	"github.com/forkbombeu/didimo/pkg/internal/apierror"
	"github.com/forkbombeu/didimo/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"

	engine "github.com/forkbombeu/didimo/pkg/template_engine"
)

func getTemplatesByFolder(folder string) ([]*os.File, error) {
	var templates []*os.File
	err := filepath.Walk(os.Getenv("ROOT_DIR")+"/config_templates/"+folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		templates = append(templates, file)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return templates, nil
}


func HandleGetConfigsTemplates(app core.App) routing.HandlerFunc {
	return func(e *core.RequestEvent) error {
		testId := e.Request.URL.Query().Get("test_id")
		if testId == "" {
			testId = "OpenID4VP_Wallet/OpenID_Foundation"
		}
		files, err := getTemplatesByFolder(testId)
		if err != nil {
			return apierror.New(http.StatusBadRequest, "request.file.read", "Error reading test suite folder", err.Error())
		}
		var variants []string
		for _, file := range files {
			variants = append(variants, p.Base(file.Name()))
		}
		return e.JSON(http.StatusOK, map[string]interface{}{
			"variants": variants,
		})
	}
}

type GetPlaceholdersByFilenamesRequestInput struct {
	TestID    string   `json:"test_id"`
	Filenames []string `json:"filenames"`
}

func HandlePlaceholdersByFilenames(app core.App) routing.HandlerFunc {
	return func(e *core.RequestEvent) error {
		requestPayload, err := routing.GetValidatedInput[GetPlaceholdersByFilenamesRequestInput](e)
		if err != nil {
			return err
		}

		if requestPayload.TestID == "" {
			requestPayload.TestID = "OpenID4VP_Wallet/OpenID_Foundation"
		}

		if len(requestPayload.Filenames) == 0 {
			return apierror.New(http.StatusBadRequest, "request.validation", "filenames are required", "filenames are required")
		}

		var files []io.Reader
		for _, filename := range requestPayload.Filenames {
			filePath := filepath.Join(os.Getenv("ROOT_DIR"), "config_templates", requestPayload.TestID, filename)
			file, err := os.Open(filePath)
			if err != nil {
				return apierror.New(http.StatusBadRequest, "request.file.open", "Error opening file: "+filename, err.Error())
			}
			defer file.Close()
			files = append(files, file)
		}

		placeholders, err := engine.GetPlaceholders(files, requestPayload.Filenames)
		if err != nil {
			return apierror.New(http.StatusBadRequest, "request.placeholders", "Error getting placeholders", err.Error())
		}

		return e.JSON(http.StatusOK, placeholders)
	}
}
