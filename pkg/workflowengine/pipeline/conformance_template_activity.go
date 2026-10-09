// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// ConformanceTemplateActivity reads a conformance check's config and StepCI
// templates outside workflow code.
type ConformanceTemplateActivity struct {
	workflowengine.BaseActivity
}

// ConformanceTemplateActivityPayload names the conformance check to resolve.
type ConformanceTemplateActivityPayload struct {
	CheckID string `json:"check_id" validate:"required"`
	StepID  string `json:"step_id"`
}

// NewConformanceTemplateActivity returns the "Resolve conformance check template" activity.
func NewConformanceTemplateActivity() *ConformanceTemplateActivity {
	return &ConformanceTemplateActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Resolve conformance check template",
		},
	}
}

// Name returns the registered activity name.
func (a *ConformanceTemplateActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute resolves the check's config template, suite extras and StepCI template.
func (a *ConformanceTemplateActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[ConformanceTemplateActivityPayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	resolved, err := resolveConformanceTemplate(payload.CheckID, payload.StepID)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}
	return workflowengine.ActivityResult{Output: resolved}, nil
}
