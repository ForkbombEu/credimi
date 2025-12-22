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
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

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

// Execute performs an HTTP request based on the provided configuration and payload.
// It supports GET, POST, PUT, DELETE methods and can handle query parameters, headers, and body.
// The result includes the status code, headers, and body of the response.
// It returns an error if the request fails or if the response status code is not 2xx.
// The timeout for the request can be configured in seconds.
func (a *HTTPActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[HTTPActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	url := payload.URL
	if payload.QueryParams != nil {
		parsedURL, err := URL.Parse(TrimInput(url))
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
			return result, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s': %v", errCode.Description, err),
				url,
			)
		}

		// Add query parameters
		query := parsedURL.Query()
		for key, value := range payload.QueryParams {
			query.Add(key, value)
		}
		parsedURL.RawQuery = query.Encode()
		url = parsedURL.String() // Update the URL with query parameters
	}

	timeout := 10 * time.Second
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
			return result, a.NewActivityError(
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
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.Method,
			url,
			body,
		)
	}

	for k, v := range payload.Headers {
		req.Header.Set(k, v)
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			req,
		)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ReadFromReaderFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			resp.Body,
		)
	}

	var output any
	if err := json.Unmarshal(respBody, &output); err != nil {
		// if not JSON, return as string
		output = string(respBody)
	}

	if payload.ExpectedStatus != 0 {
		if resp.StatusCode != payload.ExpectedStatus {
			errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
			return result, a.NewActivityError(
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
