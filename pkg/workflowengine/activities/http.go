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
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	method := input.Config["method"]
	url := input.Config["url"]
	if queryParams, ok := input.Payload["query_params"].(map[string]any); ok {
		parsedURL, err := URL.Parse(url)
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
		for key, value := range queryParams {
			if strValue, ok := value.(string); ok {
				query.Add(key, strValue)
			}
		}

		parsedURL.RawQuery = query.Encode()
		url = parsedURL.String() // Update the URL with query parameters
	}
	if method == "" || url == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'method' and 'url' must be provided", errCode.Description),
		)
	}
	timeout := 10 * time.Second
	if tStr, ok := input.Config["timeout"]; ok {
		if t, err := strconv.Atoi(tStr); err == nil {
			timeout = time.Duration(t) * time.Second
		}
	}
	headers := map[string]string{}
	if rawHeaders, ok := input.Payload["headers"].(map[string]any); ok {
		for k, v := range rawHeaders {
			if vs, ok := v.(string); ok {
				headers[k] = vs
			}
		}
	}
	var body io.Reader
	if input.Payload["body"] != nil {
		jsonBody, err := json.Marshal(input.Payload["body"])
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
			return result, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s for request body: %v", errCode.Description, err),
				input.Payload["body"],
			)
		}
		body = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CreateHTTPRequestFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			method,
			url,
			body,
		)
	}

	for k, v := range headers {
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

	if input.Payload["expected_status"] != nil {
		expectedStatus := int(input.Payload["expected_status"].(float64))
		if resp.StatusCode != expectedStatus {
			errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
			return result, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf(
					"%s: expected '%d', got '%d'",
					errCode.Description,
					expectedStatus,
					resp.StatusCode,
				),
				resp.StatusCode,
				expectedStatus,
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
