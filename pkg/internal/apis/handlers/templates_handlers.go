// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	engine "github.com/forkbombeu/credimi/pkg/templateengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var TemplateRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/template",
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/placeholders",
			Handler:       HandlePlaceholdersByFilenames,
			RequestSchema: GetPlaceholdersByFilenamesRequestInput{},
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

type GetPlaceholdersByFilenamesRequestInput struct {
	TestID    string   `json:"test_id"`
	Filenames []string `json:"filenames"`
}

func HandlePlaceholdersByFilenames() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		requestPayload, err := routing.GetValidatedInput[GetPlaceholdersByFilenamesRequestInput](e)
		if err != nil {
			return err
		}

		if len(requestPayload.Filenames) == 0 {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"filenames are required",
				"filenames are required",
			)
		}

		templatesDir := filepath.Join(os.Getenv("ROOT_DIR"), "config_templates")
		root, err := os.OpenRoot(templatesDir)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request.file.open",
				"templates unavailable",
				"templates unavailable",
			)
		}
		defer root.Close()

		var files []io.Reader
		for _, filename := range requestPayload.Filenames {
			if !strings.Contains(filename, "/") {
				continue
			}
			rel := filepath.Join(requestPayload.TestID, filename)
			if !filepath.IsLocal(rel) {
				return apierror.New(
					http.StatusBadRequest,
					"request.validation",
					"invalid filename",
					"invalid filename",
				)
			}
			file, err := root.Open(rel)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"request.file.open",
					"Error opening file: "+filename,
					"unable to open template file",
				)
			}
			defer file.Close()
			files = append(files, file)
		}

		placeholders, err := engine.GetPlaceholders(files, requestPayload.Filenames)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.placeholders",
				"Error getting placeholders",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, placeholders)
	}
}
