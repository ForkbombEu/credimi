// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

const FCAFResolveExecutionPlanActivityName = "Resolve FCAF execution plan"

type FCAFResolveExecutionPlanActivity struct {
	workflowengine.BaseActivity
	catalogLoader func(root string) (*catalog.Catalog, error)
}

type FCAFResolveExecutionPlanActivityInput struct {
	TestIDs     []string       `json:"test_ids"               yaml:"test_ids"`
	Suite       string         `json:"suite,omitempty"        yaml:"suite,omitempty"`
	CatalogRoot string         `json:"catalog_root,omitempty" yaml:"catalog_root,omitempty"`
	Runtime     map[string]any `json:"runtime,omitempty"      yaml:"runtime,omitempty"`
}

type FCAFResolveExecutionPlanActivityOutput struct {
	SelectedTests             []string                     `json:"selected_tests"`
	PipelinePreconditions     []dsl.PreconditionDefinition `json:"pipeline_preconditions"`
	TestPipelinePreconditions map[string][]string          `json:"test_pipeline_preconditions"`
}

func NewFCAFResolveExecutionPlanActivity() *FCAFResolveExecutionPlanActivity {
	return &FCAFResolveExecutionPlanActivity{
		BaseActivity:  workflowengine.BaseActivity{Name: FCAFResolveExecutionPlanActivityName},
		catalogLoader: catalog.Load,
	}
}

func (a *FCAFResolveExecutionPlanActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *FCAFResolveExecutionPlanActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[FCAFResolveExecutionPlanActivityInput](
		input.Payload,
	)
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

	selected, err := cat.ResolveSelectedTests(payload.TestIDs, suite, payload.Runtime)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
			Message: err.Error(),
		})
	}
	preconditions, err := collectPipelinePreconditions(cat, selected)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
			Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
			Message: err.Error(),
		})
	}
	testPipelines := make(map[string][]string, len(selected))
	for _, testID := range selected {
		required, collectErr := collectPipelinePreconditions(cat, []string{testID})
		if collectErr != nil {
			return workflowengine.ActivityResult{}, a.NewActivityError(workflowengine.ActivityError{
				Code:    errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Summary: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
				Message: collectErr.Error(),
			})
		}
		for _, precondition := range required {
			testPipelines[testID] = append(testPipelines[testID], precondition.ID)
		}
	}

	return workflowengine.ActivityResult{
		Output: FCAFResolveExecutionPlanActivityOutput{
			SelectedTests:             selected,
			PipelinePreconditions:     preconditions,
			TestPipelinePreconditions: testPipelines,
		},
	}, nil
}
