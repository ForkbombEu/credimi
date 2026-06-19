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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"golang.org/x/net/html"
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

	QueryParams    map[string]string     `json:"query_params,omitempty"    yaml:"query_params,omitempty"`
	Timeout        string                `json:"timeout,omitempty"         yaml:"timeout,omitempty"`
	Headers        map[string]string     `json:"headers,omitempty"         yaml:"headers,omitempty"`
	Body           any                   `json:"body,omitempty"            yaml:"body,omitempty"`
	ExpectedStatus interface{}           `json:"expected_status,omitempty" yaml:"expected_status,omitempty"`
	Outputs        map[string]OutputRule `json:"outputs,omitempty"         yaml:"outputs,omitempty"`
}

type OutputRule struct {
	XPath    string `json:"xpath,omitempty"    yaml:"xpath,omitempty"`
	Selector string `json:"selector,omitempty" yaml:"selector,omitempty"`
	Cookie   string `json:"cookie,omitempty"   yaml:"cookie,omitempty"`
	Regex    string `json:"regex,omitempty"    yaml:"regex,omitempty"`
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
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: err.Error(),
					Details: map[string]any{"url": url},
				},
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
		if bodyStr, ok := payload.Body.(string); ok {
			body = bytes.NewBufferString(bodyStr)
		} else {
			jsonBody, err := json.Marshal(payload.Body)
			if err != nil {
				errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
				return result, act.NewActivityError(
					workflowengine.ActivityError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: err.Error(),
						Details: map[string]any{"body": payload.Body},
					},
				)
			}
			body = bytes.NewBuffer(jsonBody)
		}
	}

	req, err := http.NewRequestWithContext(ctx, payload.Method, url, body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CreateHTTPRequestFailed]
		return result, act.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{
					"method": payload.Method,
					"url":    url,
					"body":   payload.Body,
				},
			},
		)
	}

	for k, v := range payload.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range injectedHeaders {
		req.Header.Set(k, v)
	}
	if body != nil && req.Header.Get(workflowengine.HTTPHeaderContentType) == "" {
		req.Header.Set(workflowengine.HTTPHeaderContentType, workflowengine.MIMEApplicationJSON)
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
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{"request": reqSnap},
			},
		)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ReadFromReaderFailed]
		return result, act.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{
					"status":  resp.StatusCode,
					"headers": resp.Header,
				},
			},
		)
	}

	var output any
	if err := json.Unmarshal(respBody, &output); err != nil {
		output = string(respBody)
	}

	output, result.Secrets, err = splitSecretsFromOutput(output)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		return result, act.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf("failed to decode response secrets: %v", err),
			},
		)
	}

	if err := validateExpectedStatus(
		resp.StatusCode,
		payload.ExpectedStatus,
		output,
		act,
	); err != nil {
		return result, err
	}

	outputValues, err := extractOutputRules(resp, respBody, payload.Outputs)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		return result, act.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf("failed to extract outputs: %v", err),
			},
		)
	}
	resultMap := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    output,
	}
	for key, value := range outputValues {
		resultMap[key] = value
	}

	result.Output = resultMap
	return result, nil
}

func splitSecretsFromOutput(output any) (any, map[string]any, error) {
	outputMap, ok := output.(map[string]any)
	if !ok {
		return output, nil, nil
	}

	rawSecrets, ok := outputMap["secrets"]
	if !ok {
		return output, nil, nil
	}

	secrets, err := decodeSecretsMap(rawSecrets)
	if err != nil {
		return nil, nil, fmt.Errorf("decode response secrets: %w", err)
	}

	cleanOutput := make(map[string]any, len(outputMap)-1)
	for key, value := range outputMap {
		if key == "secrets" {
			continue
		}
		cleanOutput[key] = value
	}

	return cleanOutput, secrets, nil
}

func decodeSecretsMap(raw any) (map[string]any, error) {
	switch values := raw.(type) {
	case map[string]any:
		return values, nil
	case map[string]string:
		secrets := make(map[string]any, len(values))
		for key, value := range values {
			secrets[key] = value
		}
		return secrets, nil
	default:
		return nil, fmt.Errorf("secrets is %T, expected map", raw)
	}
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

func extractOutputRules(
	resp *http.Response,
	body []byte,
	rules map[string]OutputRule,
) (map[string]any, error) {
	if len(rules) == 0 {
		return nil, nil
	}

	if err := validateOutputRules(rules); err != nil {
		return nil, err
	}

	results := make(map[string]any)
	bodyStr := string(body)
	contentType := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html")

	var doc *html.Node
	var docErr error

	for name, rule := range rules {
		var value any
		var err error

		switch {
		case rule.XPath != "":
			if !isHTML {
				return nil, fmt.Errorf("xpath requires HTML content, got %s", contentType)
			}
			if doc == nil && docErr == nil {
				doc, docErr = html.Parse(strings.NewReader(bodyStr))
			}
			if docErr != nil {
				err = docErr
				break
			}
			node := htmlquery.FindOne(doc, rule.XPath)
			if node != nil {
				value = htmlquery.InnerText(node)
			} else {
				value = ""
			}

		case rule.Selector != "":
			if !isHTML {
				return nil, fmt.Errorf("selector requires HTML content, got %s", contentType)
			}
			if doc == nil && docErr == nil {
				doc, docErr = html.Parse(strings.NewReader(bodyStr))
			}
			if docErr != nil {
				err = docErr
				break
			}
			query := goquery.NewDocumentFromNode(doc)
			selection := query.Find(rule.Selector)
			if selection.Length() > 0 {
				value = selection.Text()
			} else {
				value = ""
			}

		case rule.Cookie != "":
			for _, c := range resp.Cookies() {
				if c.Name == rule.Cookie {
					value = c.Value
					break
				}
			}
			if value == nil {
				value = ""
			}

		case rule.Regex != "":
			re, compileErr := regexp.Compile(rule.Regex)
			if compileErr != nil {
				return nil, fmt.Errorf("invalid regex for %s: %w", name, compileErr)
			}
			matches := re.FindStringSubmatch(bodyStr)
			switch {
			case len(matches) > 1:
				value = matches[1]
			case len(matches) > 0:
				value = matches[0]
			default:
				value = ""
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", name, err)
		}
		results[name] = value
	}

	return results, nil
}

func validateOutputRules(rules map[string]OutputRule) error {
	for name, rule := range rules {
		ruleCount := 0
		if rule.XPath != "" {
			ruleCount++
		}
		if rule.Selector != "" {
			ruleCount++
		}
		if rule.Cookie != "" {
			ruleCount++
		}
		if rule.Regex != "" {
			ruleCount++
		}

		if ruleCount == 0 {
			return fmt.Errorf(
				"output rule '%s' must specify one of: xpath, selector, cookie, or regex",
				name,
			)
		}

		if ruleCount > 1 {
			return fmt.Errorf(
				"output rule '%s' cannot specify multiple extraction methods (found: xpath=%t, selector=%t, cookie=%t, regex=%t)",
				name,
				rule.XPath != "",
				rule.Selector != "",
				rule.Cookie != "",
				rule.Regex != "",
			)
		}
	}
	return nil
}

func validateExpectedStatus(
	statusCode int,
	expectedStatus interface{},
	output any,
	act *workflowengine.BaseActivity,
) error {
	if expectedStatus == nil {
		return nil
	}

	switch v := expectedStatus.(type) {
	case float64:
		if v == 0 {
			return nil
		}
		if statusCode != int(v) {
			errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
			return act.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf("expected %d, got %d", int(v), statusCode),
					Details: map[string]any{"output": output},
				},
			)
		}
		return nil

	case int:
		if v == 0 {
			return nil
		}
		if statusCode != v {
			errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
			return act.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf("expected %d, got %d", v, statusCode),
					Details: map[string]any{"output": output},
				},
			)
		}
		return nil

	case string:
		trimmedPattern := strings.Trim(v, "/")
		matched, err := regexp.MatchString(trimmedPattern, strconv.Itoa(statusCode))
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
			return act.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf("invalid regex pattern for expected_status: %v", err),
					Details: map[string]any{"expected_status": v},
				},
			)
		}
		if !matched {
			errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
			return act.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf(
						"expected pattern '/%s/', got %d",
						trimmedPattern,
						statusCode,
					),
					Details: map[string]any{"output": output},
				},
			)
		}
		return nil

	default:
		errCode := errorcodes.Codes[errorcodes.UnexpectedHTTPStatusCode]
		return act.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf(
					"expected_status must be integer or regex string, got %T",
					expectedStatus,
				),
				Details: map[string]any{"expected_status": expectedStatus},
			},
		)
	}
}
