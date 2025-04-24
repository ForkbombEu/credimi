// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package pb provides internal utilities and routes for managing configuration templates
// and placeholders within the application.
package pb

import (
	"io"
	"net/http"
	"os"
	p "path"
	"path/filepath"

	"github.com/forkbombeu/credimi/pkg/templateengine"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
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

// RouteGetConfigsTemplates sets up a GET route to retrieve configuration templates by folder.
func RouteGetConfigsTemplates(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/conformance-checks/configs/get-configs-templates", func(e *core.RequestEvent) error {
			checkID := e.Request.URL.Query().Get("test_id")
			if checkID == "" {
				checkID = "OpenID4VP_Wallet/OpenID_Foundation"
			}
			files, err := getTemplatesByFolder(checkID)
			if err != nil {
				return apis.NewBadRequestError("Error reading test suite folder", err)
			}
			var variants []string
			for _, file := range files {
				variants = append(variants, p.Base(file.Name()))
			}
			return e.JSON(http.StatusOK, map[string]interface{}{
				"variants": variants,
			})
		})
		return se.Next()
	})
}

// RoutePostPlaceholdersByFilenames sets up a POST route to retrieve placeholders by filenames.
func RoutePostPlaceholdersByFilenames(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/conformance-checks/configs/placeholders-by-filenames", func(e *core.RequestEvent) error {
			var requestPayload struct {
				TestID    string   `json:"test_id"`
				Filenames []string `json:"filenames"`
			}

			if err := e.BindBody(&requestPayload); err != nil {
				return e.BadRequestError("Failed to read request data", err)
			}

			if requestPayload.TestID == "" {
				requestPayload.TestID = "OpenID4VP_Wallet/OpenID_Foundation"
			}

			if len(requestPayload.Filenames) == 0 {
				return apis.NewBadRequestError("filenames are required", nil)
			}

			var files []io.Reader
			for _, filename := range requestPayload.Filenames {
				filePath := filepath.Join(os.Getenv("ROOT_DIR"), "config_templates", requestPayload.TestID, filename)
				file, err := os.Open(filePath)
				if err != nil {
					return apis.NewBadRequestError("Error opening file: "+filename, err)
				}
				defer file.Close()
				files = append(files, file)
			}

			placeholders, err := templateengine.GetPlaceholders(files, requestPayload.Filenames)
			if err != nil {
				return apis.NewBadRequestError("Error getting placeholders", err)
			}

			return e.JSON(http.StatusOK, placeholders)
		})
		return se.Next()
	})
}
