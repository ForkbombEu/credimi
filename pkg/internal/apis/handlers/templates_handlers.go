// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"io"
	"net/http"
	"os"

	// p "path"
	"path/filepath"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"

	engine "github.com/forkbombeu/credimi/pkg/templateengine"
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

func HandleGetConfigsTemplates() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		configs := walkConfigTemplates()
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

type StandardMetadata struct {
	UID           string              `json:"uid"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	StandardURL   string              `json:"standard_url"`
	LatestUpdate  string              `json:"latest_update"`
	ExternalLinks map[string][]string `json:"external_links"`
}

type VersionMetadata struct {
	UID              string `json:"uid"`
	Name             string `json:"name"`
	LatestUpdate     string `json:"latest_update"`
	SpecificationURL string `json:"specification_url"`
}

type SuiteMetadata struct {
	UID         string `json:"uid"`
	Name        string `json:"name"`
	Homepage    string `json:"homepage"`
	Repository  string `json:"repository"`
	Help        string `json:"help"`
	Description string `json:"description"`
}

type Suite struct {
	Metadata SuiteMetadata `json:"metadata"`
	Files    []string      `json:"files"`
}

type Version struct {
	Version VersionMetadata `json:"version"`
	Suites  []Suite         `json:"suites"`
}

type Standard struct {
	Standard StandardMetadata `json:"standard"`
	Versions []Version        `json:"versions"`
}

type Standards []Standard

func walkConfigTemplates() Standards {
	return Standards{
		Standard{
			Standard: StandardMetadata{
				UID:          "openid4vp",
				Name:         "OpenID4VP Wallet",
				Description:  "OpenID for Verifiable Credential Issuance",
				StandardURL:  "https://openid.net/specs/openid-4-verifiable-presentations-1_0-24.html",
				LatestUpdate: "2024-02-08",
				ExternalLinks: map[string][]string{
					"reference": {},
				},
			},
			Versions: []Version{
				{
					Version: VersionMetadata{
						UID:              "draft-24",
						Name:             "Draft 13",
						LatestUpdate:     "2024-02-08",
						SpecificationURL: "https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0-13.html",
					},
					Suites: []Suite{
						{
							Metadata: SuiteMetadata{
								UID:         "ewc",
								Name:        "OpenID Foundation Conformance Suite",
								Homepage:    "https://openid.net/certification/about-conformance-suite/",
								Repository:  "https://gitlab.com/openid/conformance-suite",
								Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
							},
							Files: []string{"ewc_file1.json", "ewc_file2.json"},
						},
						{
							Metadata: SuiteMetadata{
								UID:         "openid_conformance_suite",
								Name:        "OpenID Foundation Conformance Suite",
								Homepage:    "https://openid.net/certification/about-conformance-suite/",
								Repository:  "https://gitlab.com/openid/conformance-suite",
								Help:        "https://openid.net/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for OIDF’s OpenID Connect, FAPI & FAPI-CIBA Profiles",
							},
							Files: []string{"conformance_file1.json", "conformance_file2.json"},
						},
						{
							Metadata: SuiteMetadata{
								UID:         "vuota_conformance_suite",
								Name:        "Vuota Conformance Suite",
								Homepage:    "https://vuota.com/certification/about-conformance-suite/",
								Repository:  "https://gitlab.com/vuota/conformance-suite",
								Help:        "https://vuota.com/certification/conformance-testing-for-openid-for-verifiable-presentations/",
								Description: "Conformance suite for Vuota",
							},
							Files: []string{},
						},
					},
				},
			},
		},
	}
}
