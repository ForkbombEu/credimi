// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

const (
	FCAFValidationActivityName       = "Run FCAF validation"
	DefaultFCAFValidationCatalogRoot = "config_templates/fcaf/wallet_solution/relying_party"
)

type FCAFValidationActivityInput struct {
	TestID      string         `json:"test_id,omitempty"          yaml:"test_id,omitempty"`
	TestIDs     []string       `json:"test_ids,omitempty"         yaml:"test_ids,omitempty"`
	Suite       string         `json:"suite,omitempty"            yaml:"suite,omitempty"`
	CatalogRoot string         `json:"catalog_root,omitempty"     yaml:"catalog_root,omitempty"`
	Pipeline    map[string]any `json:"pipeline_outputs,omitempty" yaml:"pipeline_outputs,omitempty"`
	Runtime     map[string]any `json:"runtime,omitempty"          yaml:"runtime,omitempty"`
}

type FCAFValidationActivityOutput struct {
	Report engine.Report `json:"report"`
}

type FCAFValidationActivity struct {
	workflowengine.BaseActivity
	catalogLoader func(root string) (*catalog.Catalog, error)
}

func NewFCAFValidationActivity() *FCAFValidationActivity {
	return &FCAFValidationActivity{
		BaseActivity:  workflowengine.BaseActivity{Name: FCAFValidationActivityName},
		catalogLoader: catalog.Load,
	}
}

func (a *FCAFValidationActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *FCAFValidationActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFValidationActivityInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	testIDs := normalizeValidationTestIDs(payload)
	if len(testIDs) == 0 {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(
			fmt.Errorf("test_id or test_ids is required"),
		)
	}
	if len(payload.Pipeline) == 0 {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(
			fmt.Errorf("pipeline_outputs is required"),
		)
	}

	catalogRoot := payload.CatalogRoot
	if catalogRoot == "" {
		catalogRoot = resolveDefaultFCAFValidationCatalogRoot()
	}
	suite := payload.Suite
	if suite == "" {
		suite = "wallet_solution/relying_party"
	}
	loader := a.catalogLoader
	if loader == nil {
		loader = catalog.Load
	}
	cat, err := loader(catalogRoot)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
			Message: err.Error(),
		})
	}
	fcafEngine, err := engine.New(nil)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
			Message: fmt.Sprintf("create fcaf engine: %v", err),
		})
	}
	bundle := evidence.Bundle{PipelineOutputs: payload.Pipeline, Runtime: payload.Runtime}
	report, err := fcafEngine.ExecuteCatalog(
		ctx,
		cat,
		testIDs,
		suite,
		payload.Runtime,
		bundle,
	)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
			Message: fmt.Sprintf("execute fcaf engine: %v", err),
		})
	}
	report.PopulateDerivedViews()
	return workflowengine.ActivityResult{
		Output: FCAFValidationActivityOutput{Report: report.PublicReport()},
	}, nil
}

func normalizeValidationTestIDs(payload FCAFValidationActivityInput) []string {
	ids := make([]string, 0, len(payload.TestIDs))
	seen := make(map[string]struct{}, len(payload.TestIDs))
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, id := range payload.TestIDs {
		add(id)
	}
	add(payload.TestID)
	return ids
}

func resolveDefaultFCAFValidationCatalogRoot() string {
	if _, err := os.Stat(DefaultFCAFValidationCatalogRoot); err == nil {
		return DefaultFCAFValidationCatalogRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return DefaultFCAFValidationCatalogRoot
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, DefaultFCAFValidationCatalogRoot)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return DefaultFCAFValidationCatalogRoot
}
