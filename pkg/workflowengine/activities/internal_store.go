// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/activity"
)

const internalStoreErrorBodyLimit = 1 << 10

// internalStoreRetryWaits are the waits before the second and third attempt.
var internalStoreRetryWaits = []time.Duration{5 * time.Second, 15 * time.Second}

// postInternalJSON POSTs body to a trusted internal Credimi route and returns the response
// body of a 200 answer. It retries network errors and 5xx answers. Unlike the
// internal-http-request activity, its errors never carry the request body, so a failed
// store of a large document does not copy it into history.
func postInternalJSON(
	ctx context.Context,
	baseURL string,
	pathSegments []string,
	body any,
	timeout time.Duration,
) ([]byte, error) {
	apiKey := strings.TrimSpace(os.Getenv("CREDIMI_INTERNAL_ADMIN_KEY"))
	if apiKey == "" {
		return nil, errors.New("CREDIMI_INTERNAL_ADMIN_KEY is required")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint := utils.JoinURL(baseURL, pathSegments...)
	path := "/" + strings.Join(pathSegments, "/")
	client := &http.Client{Timeout: timeout}

	for attempt := 0; ; attempt++ {
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, attempt)
		}
		respBody, retry, err := postInternalJSONOnce(ctx, client, endpoint, apiKey, data)
		if err == nil {
			return respBody, nil
		}
		if !retry || attempt >= len(internalStoreRetryWaits) {
			return nil, fmt.Errorf("POST %s: %w", path, err)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("POST %s: %w", path, ctx.Err())
		case <-time.After(internalStoreRetryWaits[attempt]):
		}
	}
}

// postInternalJSONOnce makes one attempt and reports whether a failure is worth retrying.
func postInternalJSONOnce(
	ctx context.Context,
	client *http.Client,
	endpoint, apiKey string,
	data []byte,
) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set(workflowengine.HTTPHeaderContentType, workflowengine.MIMEApplicationJSON)
	req.Header.Set("Credimi-Api-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusOK {
		return respBody, false, nil
	}
	excerpt := respBody
	if len(excerpt) > internalStoreErrorBodyLimit {
		excerpt = excerpt[:internalStoreErrorBodyLimit]
	}
	return nil, resp.StatusCode >= http.StatusInternalServerError,
		fmt.Errorf("status %d: %s", resp.StatusCode, excerpt)
}
