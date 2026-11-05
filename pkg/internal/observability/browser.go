// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package observability

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/time/rate"
)

// BrowserTracesPath receives OTLP/JSON traces from the webapp's browser SDK.
// Browsers cannot hold the collector credentials, so Credimi forwards the
// batches with its own.
const BrowserTracesPath = "/api/otlp/v1/traces"

// BrowserServiceName is the service.name every forwarded batch is given:
// anyone can post here, so browser data must never pass for backend data.
const BrowserServiceName = "credimi-browser"

const (
	maxBrowserBatchBytes = 512 << 10
	// An exporting tab sends a batch every few seconds at most.
	browserRate           = rate.Limit(2)
	browserBurst          = 20
	maxTrackedLimiterKeys = 10000
)

// ipLimiter rate-limits per key (client IP or API key owner).
type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

func (l *ipLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	limiter, ok := l.limiters[key]
	if !ok {
		if len(l.limiters) >= maxTrackedLimiterKeys {
			l.limiters = map[string]*rate.Limiter{}
		}
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[key] = limiter
	}
	return limiter.Allow()
}

// otlpHeaders parses OTEL_EXPORTER_OTLP_HEADERS: comma-separated key=value
// pairs with percent-encoded values.
func otlpHeaders(raw string) http.Header {
	headers := http.Header{}
	for _, pair := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok || key == "" {
			continue
		}
		if decoded, err := url.QueryUnescape(value); err == nil {
			value = decoded
		}
		headers.Set(key, value)
	}
	return headers
}

// forceServiceName replaces the service.name of every resource in an OTLP/JSON
// traces payload.
func forceServiceName(body []byte, service string) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	resourceSpans, ok := payload["resourceSpans"].([]any)
	if !ok {
		return nil, errors.New("resourceSpans missing")
	}
	serviceName := map[string]any{
		"key":   "service.name",
		"value": map[string]any{"stringValue": service},
	}
	for _, item := range resourceSpans {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("invalid resourceSpans entry")
		}
		resource, _ := entry["resource"].(map[string]any)
		if resource == nil {
			resource = map[string]any{}
			entry["resource"] = resource
		}
		attributes, _ := resource["attributes"].([]any)
		kept := []any{serviceName}
		for _, attr := range attributes {
			if kv, ok := attr.(map[string]any); ok && kv["key"] != "service.name" {
				kept = append(kept, kv)
			}
		}
		resource["attributes"] = kept
	}
	return json.Marshal(payload)
}

func browserTracesHandler() func(*core.RequestEvent) error {
	target := strings.TrimRight(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "/") + "/v1/traces"
	headers := otlpHeaders(os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))
	limiter := &ipLimiter{
		limiters: map[string]*rate.Limiter{},
		rate:     browserRate,
		burst:    browserBurst,
	}
	client := &http.Client{Timeout: 10 * time.Second}

	return func(e *core.RequestEvent) error {
		if !limiter.allow(e.RealIP()) {
			return e.TooManyRequestsError("", nil)
		}
		if !strings.HasPrefix(e.Request.Header.Get("Content-Type"), "application/json") {
			return e.Error(http.StatusUnsupportedMediaType, "OTLP/JSON only", nil)
		}
		body, err := io.ReadAll(
			http.MaxBytesReader(e.Response, e.Request.Body, maxBrowserBatchBytes),
		)
		if err != nil {
			return e.Error(http.StatusRequestEntityTooLarge, "batch too large", nil)
		}
		body, err = forceServiceName(body, BrowserServiceName)
		if err != nil {
			return e.BadRequestError("invalid OTLP/JSON traces", nil)
		}

		req, err := http.NewRequestWithContext(
			e.Request.Context(),
			http.MethodPost,
			target,
			bytes.NewReader(body),
		)
		if err != nil {
			return err
		}
		req.Header = headers.Clone()
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return e.Error(http.StatusBadGateway, "collector unreachable", nil)
		}
		defer resp.Body.Close()
		return e.Stream(resp.StatusCode, "application/json", resp.Body)
	}
}
