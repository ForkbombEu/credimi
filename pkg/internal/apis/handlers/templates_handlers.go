// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	engine "github.com/forkbombeu/credimi/pkg/templateengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"gopkg.in/yaml.v3"
)

var TemplateRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/template",
	Routes: []routing.RouteDefinition{
		{
			Method:  "GET",
			Path:    "/blueprints",
			Handler: HandleGetConfigsTemplates,
		},
		{
			Method:        "POST",
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

var templatesDir = path.Join(os.Getenv("ROOT_DIR"), "config_templates")

func HandleGetConfigsTemplates() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		configs, err := walkConfigTemplates(templatesDir)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, configs)
	}
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

		var files []io.Reader
		for _, filename := range requestPayload.Filenames {
			if !strings.Contains(filename, "/") {
				continue
			}
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

type StandardMetadata struct {
	UID           string              `json:"uid"            yaml:"uid"`
	Name          string              `json:"name"           yaml:"name"`
	Description   string              `json:"description"    yaml:"description"`
	StandardURL   string              `json:"standard_url"   yaml:"standard_url"`
	LatestUpdate  string              `json:"latest_update"  yaml:"latest_update"`
	ExternalLinks map[string][]string `json:"external_links" yaml:"external_links"`
	Disabled      bool                `json:"disabled"       yaml:"disabled"`
}

type VersionMetadata struct {
	UID              string `json:"uid"               yaml:"uid"`
	Name             string `json:"name"              yaml:"name"`
	LatestUpdate     string `json:"latest_update"     yaml:"latest_update"`
	SpecificationURL string `json:"specification_url" yaml:"specification_url"`
}

type SuiteMetadata struct {
	UID         string `json:"uid"         yaml:"uid"`
	Name        string `json:"name"        yaml:"name"`
	Homepage    string `json:"homepage"    yaml:"homepage"`
	Repository  string `json:"repository"  yaml:"repository"`
	Help        string `json:"help"        yaml:"help"`
	Description string `json:"description" yaml:"description"`
}

type Suite struct {
	SuiteMetadata
	Files []string `json:"files" yaml:"files"`
}

type Version struct {
	VersionMetadata
	Suites []Suite `json:"suites" yaml:"suites"`
}

type Standard struct {
	StandardMetadata
	Versions []Version `json:"versions" yaml:"versions"`
}

type Standards []Standard

func walkConfigTemplates(dir string) (Standards, error) {
	var standards = make(Standards, 0)

	readDir := func(path string) ([]os.DirEntry, error) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, apierror.New(
				http.StatusInternalServerError,
				"filesystem.readDir",
				"Failed to read directory: "+path,
				err.Error(),
			)
		}
		return entries, nil
	}

	readYaml := func(path string, out interface{}) error {
		data, err := os.ReadFile(path)
		if err != nil {
			// If file doesn't exist, just skip (not an error)
			if os.IsNotExist(err) {
				return nil
			}
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem.readFile",
				"Failed to read file: "+path,
				err.Error(),
			)
		}
		if err := yaml.Unmarshal(data, out); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"yaml.unmarshal",
				"Failed to unmarshal yaml: "+path,
				err.Error(),
			)
		}
		return nil
	}

	standardEntries, err := readDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range standardEntries {
		if !entry.IsDir() {
			continue
		}
		standardUID := entry.Name()
		standardPath := filepath.Join(dir, standardUID)

		standardMeta := StandardMetadata{UID: standardUID}
		if err := readYaml(filepath.Join(standardPath, "standard.yaml"), &standardMeta); err != nil {
			return nil, err
		}

		versionEntries, err := readDir(standardPath)
		if err != nil {
			return nil, err
		}

		var versions []Version
		for _, vEntry := range versionEntries {
			if !vEntry.IsDir() {
				continue
			}
			versionUID := vEntry.Name()
			versionPath := filepath.Join(standardPath, versionUID)

			versionMeta := VersionMetadata{UID: versionUID}
			if err := readYaml(filepath.Join(versionPath, "version.yaml"), &versionMeta); err != nil {
				return nil, err
			}

			suiteEntries, err := readDir(versionPath)
			if err != nil {
				return nil, err
			}

			var suites []Suite
			for _, sEntry := range suiteEntries {
				if !sEntry.IsDir() {
					continue
				}
				suiteUID := sEntry.Name()
				suitePath := filepath.Join(versionPath, suiteUID)

				suiteMeta := SuiteMetadata{UID: suiteUID}
				if err := readYaml(filepath.Join(suitePath, "metadata.yaml"), &suiteMeta); err != nil {
					return nil, err
				}

				fileEntries, err := readDir(suitePath)
				if err != nil {
					return nil, err
				}

				files := []string{}
				for _, f := range fileEntries {
					if !f.IsDir() && f.Name() != "metadata.yaml" {
						files = append(files, f.Name())
					}
				}

				suites = append(suites, Suite{
					SuiteMetadata: suiteMeta,
					Files:         files,
				})
			}

			versions = append(versions, Version{
				VersionMetadata: versionMeta,
				Suites:          suites,
			})
		}

		standards = append(standards, Standard{
			StandardMetadata: standardMeta,
			Versions:         versions,
		})
	}

	return standards, nil
}
