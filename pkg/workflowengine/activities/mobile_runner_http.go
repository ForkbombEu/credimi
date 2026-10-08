// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// MobileRunnerHTTPActivity calls a mobile runner at the address stored on its
// mobile_runners record, presenting that runner's own credential, through the
// runner transport (destination policy for tenant runners, Cloudflare DNS for
// quick tunnels). A run's Temporal history names the calls that leave Credimi
// for a runner.
type MobileRunnerHTTPActivity struct {
	workflowengine.BaseActivity
	app core.App
}

// MobileRunnerHTTPActivityPayload names the runner by its canonified
// identifier and the endpoint by its path; the destination address always
// comes from the runner record.
type MobileRunnerHTTPActivityPayload struct {
	Method         string            `json:"method"                    yaml:"method"                    validate:"required"`
	RunnerID       string            `json:"runner_id"                 yaml:"runner_id"                 validate:"required"`
	Path           string            `json:"path"                      yaml:"path"                      validate:"required"`
	QueryParams    map[string]string `json:"query_params,omitempty"    yaml:"query_params,omitempty"`
	Timeout        string            `json:"timeout,omitempty"         yaml:"timeout,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"         yaml:"headers,omitempty"`
	Body           any               `json:"body,omitempty"            yaml:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
}

// NewMobileRunnerHTTPActivity returns the activity; callers that only need its
// Name may pass a nil app.
func NewMobileRunnerHTTPActivity(app core.App) *MobileRunnerHTTPActivity {
	return &MobileRunnerHTTPActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Make an HTTP request to a mobile runner"},
		app:          app,
	}
}

func (a *MobileRunnerHTTPActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *MobileRunnerHTTPActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}
	payload, err := workflowengine.DecodePayload[MobileRunnerHTTPActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	runner, err := mobilerunner.ResolveRunner(a.app, payload.RunnerID)
	if err != nil {
		if errors.Is(err, mobilerunner.ErrRunnerNotFound) {
			return result, a.NewCodedError(errorcodes.RecordNotFound, false, err)
		}
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
	runnerURL := mobilerunner.RunnerURL(runner)
	if runnerURL == "" {
		return result, a.NewCodedError(
			errorcodes.MissingOrInvalidConfig,
			false,
			fmt.Errorf("mobile runner %s has no stored address", payload.RunnerID),
		)
	}
	credential, err := mobilerunner.Credential(runner)
	if err != nil {
		return result, a.NewCodedError(errorcodes.MissingOrInvalidConfig, true, err)
	}

	return executeHTTPRequest(
		ctx,
		HTTPActivityPayload{
			Method:         payload.Method,
			URL:            utils.JoinURL(runnerURL, payload.Path),
			QueryParams:    payload.QueryParams,
			Timeout:        payload.Timeout,
			Headers:        payload.Headers,
			Body:           payload.Body,
			ExpectedStatus: payload.ExpectedStatus,
		},
		map[string]string{"Credimi-Api-Key": credential},
		&a.BaseActivity,
		mobilerunner.Transport(runner),
	)
}
