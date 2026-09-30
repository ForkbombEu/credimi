// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// MobileRunnerHTTPActivity carries the internal admin credential exactly like
// InternalHTTPActivity, and additionally resolves quick-tunnel runner hostnames
// through Cloudflare DNS. It is a separate activity rather than a branch inside
// the internal one so a run's Temporal history names the calls that leave
// Credimi for a runner.
type MobileRunnerHTTPActivity struct {
	workflowengine.BaseActivity
}

type MobileRunnerHTTPActivityPayload struct {
	Method         string            `json:"method"                    yaml:"method"                    validate:"required"`
	URL            string            `json:"url"                       yaml:"url"                       validate:"required"`
	QueryParams    map[string]string `json:"query_params,omitempty"    yaml:"query_params,omitempty"`
	Timeout        string            `json:"timeout,omitempty"         yaml:"timeout,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"         yaml:"headers,omitempty"`
	Body           any               `json:"body,omitempty"            yaml:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
}

func NewMobileRunnerHTTPActivity() *MobileRunnerHTTPActivity {
	return &MobileRunnerHTTPActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Make an HTTP request to a mobile runner"},
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

	return executeInternalHTTPRequest(ctx, InternalHTTPActivityPayload{
		Method:         payload.Method,
		URL:            payload.URL,
		QueryParams:    payload.QueryParams,
		Timeout:        payload.Timeout,
		Headers:        payload.Headers,
		Body:           payload.Body,
		ExpectedStatus: payload.ExpectedStatus,
	}, &a.BaseActivity, mobilerunner.Transport(payload.URL))
}
