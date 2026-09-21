// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/conformancecatalog"
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
			Method:              http.MethodGet,
			Path:                "/blueprints",
			Handler:             HandleGetConfigsTemplates,
			ExcludedMiddlewares: []string{middlewares.RequireAuthOrAPIKeyMiddlewareID},
		},
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

func HandleGetConfigsTemplates() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		surface := e.Request.URL.Query().Get("surface")
		if surface != "" && surface != TemplateSurfaceManual && surface != TemplateSurfacePipeline {
			return apierror.New(
				http.StatusBadRequest,
				"surface",
				"invalid value for surface",
				fmt.Sprintf(
					"surface must be %q or %q",
					TemplateSurfaceManual,
					TemplateSurfacePipeline,
				),
			)
		}

		// Nested blueprints are projected from the conformance catalog snapshot
		// (boot Rebuild / LoadFromDir) — no per-request filesystem walk.
		return e.JSON(http.StatusOK, conformancecatalog.Default().Blueprints(surface))
	}
}

const (
	TemplateSurfaceManual   = conformancecatalog.SurfaceManual
	TemplateSurfacePipeline = conformancecatalog.SurfacePipeline
)

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

		var files []io.Reader
		for _, filename := range requestPayload.Filenames {
			if !strings.Contains(filename, "/") {
				continue
			}
			templatesDir := path.Join(os.Getenv("ROOT_DIR"), "config_templates")
			filePath := filepath.Join(templatesDir, requestPayload.TestID, filename)
			file, err := os.Open(filePath)
			if err != nil {
				return apierror.New(
					http.StatusBadRequest,
					"request.file.open",
					"Error opening file: "+filename,
					err.Error(),
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

// Compatibility aliases for existing handler tests and callers.
type (
	StandardMetadata = conformancecatalog.StandardMetadata
	VersionMetadata  = conformancecatalog.VersionMetadata
	SuiteMetadata    = conformancecatalog.SuiteMetadata
	Suite            = conformancecatalog.Suite
	Version          = conformancecatalog.Version
	Standard         = conformancecatalog.Standard
	Standards        = conformancecatalog.Blueprints
)
