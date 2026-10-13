// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"net/http"
	"net/url"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// openIDNetLogsBaseURL is the OpenID certification log API; tests replace it.
var openIDNetLogsBaseURL = "https://www.certification.openid.net/api/log"

// OpenIDNetLogsActivity fetches the logs of an OpenID certification run. The
// bearer token is read from OPENIDNET_TOKEN on the worker, so it never enters
// workflow history.
type OpenIDNetLogsActivity struct {
	workflowengine.BaseActivity
}

// OpenIDNetLogsPayload identifies the certification run whose logs to fetch.
type OpenIDNetLogsPayload struct {
	Rid string `json:"rid" yaml:"rid" validate:"required"`
}

// NewOpenIDNetLogsActivity returns the "Fetch OpenID certification logs" activity.
func NewOpenIDNetLogsActivity() *OpenIDNetLogsActivity {
	return &OpenIDNetLogsActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Fetch OpenID certification logs",
		},
	}
}

// Name returns the name of the OpenID certification logs activity.
func (a *OpenIDNetLogsActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute fetches the run logs; the result has the HTTP activity's shape.
func (a *OpenIDNetLogsActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[OpenIDNetLogsPayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	token := utils.GetEnvironmentVariable("OPENIDNET_TOKEN")
	if token == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return workflowengine.ActivityResult{}, a.NewNonRetryableActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "OPENIDNET_TOKEN is not set",
			},
		)
	}

	return executeHTTPRequest(
		ctx,
		HTTPActivityPayload{
			Method:         http.MethodGet,
			URL:            utils.JoinURL(openIDNetLogsBaseURL, url.PathEscape(payload.Rid)),
			QueryParams:    map[string]string{"public": "false"},
			ExpectedStatus: 200,
			Timeout:        "30",
		},
		map[string]string{"Authorization": "Bearer " + token},
		&a.BaseActivity,
		nil,
	)
}
