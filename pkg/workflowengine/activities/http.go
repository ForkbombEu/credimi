// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	URL "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

var sensitiveHeaders = map[string]struct{}{
	"authorization":   {},
	"credimi-api-key": {},
	"cookie":          {},
	"set-cookie":      {},
}

// HTTPActivity is an activity that performs an HTTP request.
type HTTPActivity struct {
	workflowengine.BaseActivity
}

// HTTPActivityPayload is the input payload for the HTTP activity.
type HTTPActivityPayload struct {
	Method string `json:"method" yaml:"method" validate:"required"`
	URL    string `json:"url"    yaml:"url"    validate:"required"`

	QueryParams    map[string]string `json:"query_params,omitempty"    yaml:"query_params,omitempty"`
	Timeout        string            `json:"timeout,omitempty"         yaml:"timeout,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"         yaml:"headers,omitempty"`
	Body           any               `json:"body,omitempty"            yaml:"body,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
}

type RequestSnapshot struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers,omitempty"`
	Body    any                 `json:"body,omitempty"`
}

func NewHTTPActivity() *HTTPActivity {
	return &HTTPActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Make an HTTP request",
		},
	}
}

// Name returns the name of the HTTP activity.
func (a *HTTPActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *HTTPActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}
	payload, err := workflowengine.DecodePayload[HTTPActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	return executeHTTPRequest(ctx, payload, nil, &a.BaseActivity)
}

func executeHTTPRequest(
	ctx context.Context,
	payload HTTPActivityPayload,
	injectedHeaders map[string]string,
	act *workflowengine.BaseActivity,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	url := payload.URL
	if payload.QueryParams != nil {
		parsedURL, err := URL.Parse(TrimInput(url))
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
			return result, act.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s': %v", errCode.Description, err),
				url,
			)
		}

		query := parsedURL.Query()
		for key, value := range payload.QueryParams {
			query.Add(key, value)
		}
		parsedURL.RawQuery = query.Encode()
		url = parsedURL.String()
	}

	timeout := 1 * time.Minute
	if payload.Timeout != "" {
		if t, err := strconv.Atoi(payload.Timeout); err == nil {
			timeout = time.Duration(t) * time.Second
		}
	}

	var body io.Reader
	if payload.Body != nil {
		jsonBody, err := json.Marshal(payload.Body)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
			return result, act.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s for request body: %v", errCode.Description, err),
				payload.Body,
			)
		}
		body = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, payload.Method, url, body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CreateHTTPRequestFailed]
		return result, act.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.Method,
			url,
			payload.Body,
		)
	}

	for k, v := range payload.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range injectedHeaders {
		req.Header.Set(k, v)
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	reqSnap := RequestSnapshot{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: redactHeaderMap(req.Header),
		Body:    payload.Body,
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed]
		return result, act.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			reqSnap,
		)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ReadFromReaderFailed]
		return result, act.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			resp.StatusCode,
			resp.Header,
		)
	}

	var output any
	if err := json.Unmarshal(respBody, &output); err != nil {
		output = string(respBody)
	}

	if payload.ExpectedStatus != 0 {
		if resp.StatusCode != payload.ExpectedStatus {
			errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
			return result, act.NewActivityError(
				errCode.Code,
				fmt.Sprintf(
					"%s: expected '%d', got '%d'",
					errCode.Description,
					payload.ExpectedStatus,
					resp.StatusCode,
				),
				output,
			)
		}
	}
	result.Output = map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    output,
	}

	return result, nil
}

func redactHeaderMap(headers http.Header) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string][]string, len(headers))
	for key, values := range headers {
		lower := strings.ToLower(key)
		if _, sensitive := sensitiveHeaders[lower]; sensitive {
			out[key] = []string{"[REDACTED]"}
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		out[key] = copied
	}
	return out
}
