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
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

const (
	FCAFAssessmentActivityName       = "Run FCAF assessment"
	DefaultFCAFAssessmentCatalogRoot = "config_templates/fcaf/wallet_solution/relying_party"
)

type FCAFAssessmentActivity struct {
	workflowengine.BaseActivity
	catalogLoader func(root string) (*catalog.Catalog, error)
	decoderCache  *evidence.DecoderCache
	nodeCache     *engine.NodeResultCache
}

type FCAFAssessmentActivityInput struct {
	TestIDs     []string        `json:"test_ids"               yaml:"test_ids"`
	Suite       string          `json:"suite,omitempty"        yaml:"suite,omitempty"`
	CatalogRoot string          `json:"catalog_root,omitempty" yaml:"catalog_root,omitempty"`
	Evidence    evidence.Bundle `json:"evidence,omitempty"     yaml:"evidence,omitempty"`
	Runtime     map[string]any  `json:"runtime,omitempty"      yaml:"runtime,omitempty"`
}

type FCAFAssessmentActivityOutput struct {
	Report engine.Report `json:"report"`
}

func NewFCAFAssessmentActivity() *FCAFAssessmentActivity {
	return &FCAFAssessmentActivity{
		BaseActivity:  workflowengine.BaseActivity{Name: FCAFAssessmentActivityName},
		catalogLoader: catalog.Load,
		decoderCache:  evidence.NewDecoderCache(8),
		nodeCache:     engine.NewNodeResultCache(256),
	}
}

func (a *FCAFAssessmentActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *FCAFAssessmentActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFAssessmentActivityInput](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}
	catalogRoot := payload.CatalogRoot
	if catalogRoot == "" {
		catalogRoot = resolveDefaultFCAFCatalogRoot()
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
	payload.Evidence.Runtime = payload.Runtime
	decoderCache := a.decoderCache
	if decoderCache == nil {
		decoderCache = evidence.NewDecoderCache(8)
	}
	fcafEngine, err := engine.NewWithCaches(nil, decoderCache.Extract, a.nodeCache)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
			Message: fmt.Sprintf("create fcaf engine: %v", err),
		})
	}
	report, err := fcafEngine.ExecuteCatalog(
		ctx,
		cat,
		payload.TestIDs,
		suite,
		payload.Runtime,
		payload.Evidence,
	)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: errorcodes.Codes[errorcodes.PipelineExecutionError].Description,
			Message: fmt.Sprintf("execute fcaf engine: %v", err),
		})
	}
	report.PopulateDerivedViews()
	publicReport := report.PublicReport()

	return workflowengine.ActivityResult{
		Output: FCAFAssessmentActivityOutput{Report: publicReport},
	}, nil
}

func resolveDefaultFCAFCatalogRoot() string {
	if _, err := os.Stat(DefaultFCAFAssessmentCatalogRoot); err == nil {
		return DefaultFCAFAssessmentCatalogRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return DefaultFCAFAssessmentCatalogRoot
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, DefaultFCAFAssessmentCatalogRoot)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return DefaultFCAFAssessmentCatalogRoot
}

func collectPipelinePreconditions(
	cat *catalog.Catalog,
	selected []string,
) ([]dsl.PreconditionDefinition, error) {
	ordered := []dsl.PreconditionDefinition{}
	seenPipelines := map[string]struct{}{}
	seenTests := map[string]struct{}{}

	var walkRef func(string) error
	var walkTest func(string) error

	walkRef = func(ref string) error {
		switch {
		case strings.HasPrefix(ref, "pipeline."):
			precondition, ok := cat.Preconditions[ref]
			if !ok {
				return fmt.Errorf("precondition %q not found", ref)
			}
			if _, ok := seenPipelines[precondition.ID]; ok {
				return nil
			}
			seenPipelines[precondition.ID] = struct{}{}
			ordered = append(ordered, precondition)
			return nil
		case strings.HasPrefix(ref, "assertion."):
			precondition, ok := cat.Preconditions[ref]
			if !ok {
				return fmt.Errorf("precondition %q not found", ref)
			}
			for _, dependency := range precondition.DependsOn {
				if err := walkRef(dependency); err != nil {
					return err
				}
			}
			return nil
		case strings.HasPrefix(ref, "test."):
			return walkTest(strings.TrimPrefix(ref, "test."))
		default:
			return fmt.Errorf("unsupported reference %q", ref)
		}
	}

	walkTest = func(testID string) error {
		if _, ok := seenTests[testID]; ok {
			return nil
		}
		seenTests[testID] = struct{}{}
		test, ok := cat.Tests[testID]
		if !ok {
			return fmt.Errorf("test %q not found", testID)
		}
		for _, ref := range test.Preconditions {
			if err := walkRef(ref.Ref); err != nil {
				return err
			}
		}
		return nil
	}

	for _, testID := range selected {
		if err := walkTest(testID); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
