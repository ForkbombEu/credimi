// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type InternalHTTPActivity struct {
	workflowengine.BaseActivity
}

type InternalHTTPAuthLevel string

const InternalHTTPAuthLevelAdmin InternalHTTPAuthLevel = "internal_admin"

type InternalHTTPActivityPayload struct {
	Method         string            `json:"method" yaml:"method" validate:"required"`
	URL            string            `json:"url" yaml:"url" validate:"required"`
	QueryParams    map[string]string `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	Timeout        string            `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Headers        map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body           any               `json:"body,omitempty" yaml:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
	AuthLevel      InternalHTTPAuthLevel `json:"auth_level,omitempty" yaml:"auth_level,omitempty"`
}

func NewInternalHTTPActivity() *InternalHTTPActivity {
	return &InternalHTTPActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Make an internal HTTP request"},
	}
}

func (a *InternalHTTPActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *InternalHTTPActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}
	payload, err := workflowengine.DecodePayload[InternalHTTPActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	authLevel := payload.AuthLevel
	if authLevel == "" {
		authLevel = InternalHTTPAuthLevelAdmin
	}
	if authLevel != InternalHTTPAuthLevelAdmin {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, fmt.Sprintf("unsupported auth level: %s", authLevel))
	}

	apiKey := strings.TrimSpace(os.Getenv("INTERNAL_ADMIN_API_KEY"))
	if apiKey == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return result, a.NewActivityError(errCode.Code, "INTERNAL_ADMIN_API_KEY is required")
	}

	httpPayload := HTTPActivityPayload{
		Method:         payload.Method,
		URL:            payload.URL,
		QueryParams:    payload.QueryParams,
		Timeout:        payload.Timeout,
		Headers:        payload.Headers,
		Body:           payload.Body,
		ExpectedStatus: payload.ExpectedStatus,
	}

	return executeHTTPRequest(ctx, httpPayload, map[string]string{"X-Api-Key": apiKey}, &a.BaseActivity)
}
