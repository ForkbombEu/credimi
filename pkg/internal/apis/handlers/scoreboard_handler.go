// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

/*
import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// OpenTelemetry-compatible trace and span structures
type OTelAttribute struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type OTelSpan struct {
	TraceID    string          `json:"traceId"`
	SpanID     string          `json:"spanId"`
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	StartTime  int64           `json:"startTimeUnixNano"`
	EndTime    int64           `json:"endTimeUnixNano"`
	Attributes []OTelAttribute `json:"attributes"`
	Status     OTelStatus      `json:"status"`
}

type OTelStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type OTelResourceSpan struct {
	Resource struct {
		Attributes []OTelAttribute `json:"attributes"`
	} `json:"resource"`
	ScopeSpans []struct {
		Scope struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"scope"`
		Spans []OTelSpan `json:"spans"`
	} `json:"scopeSpans"`
}

type OTelTracesData struct {
	ResourceSpans []OTelResourceSpan `json:"resourceSpans"`
}

// Scoreboard summary entry
type ScoreboardEntry struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"` // "wallet", "issuer", "verifier", "pipeline"
	TotalRuns    int     `json:"totalRuns"`
	SuccessCount int     `json:"successCount"`
	FailureCount int     `json:"failureCount"`
	SuccessRate  float64 `json:"successRate"`
	LastRun      string  `json:"lastRun"`
}

type ScoreboardResponse struct {
	Summary struct {
		Wallets   []ScoreboardEntry `json:"wallets"`
		Issuers   []ScoreboardEntry `json:"issuers"`
		Verifiers []ScoreboardEntry `json:"verifiers"`
		Pipelines []ScoreboardEntry `json:"pipelines"`
	} `json:"summary"`
	OTelData OTelTracesData `json:"otelData"`
}

var ScoreboardRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/my/results",
			Handler:     HandleMyResults,
			Description: "Get scoreboard results for the current user's Wallet/Issuer/Verifier/Pipelines",
		},
	},
}

var ScoreboardPublicRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/all-results",
			Handler:     HandleAllResults,
			Description: "Get scoreboard results for all Wallet/Issuer/Verifier/Pipelines",
		},
	},
}

// HandleMyResults returns results for the current user's entities
func HandleMyResults() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}

		userID := e.Auth.Id
		orgID, err := GetUserOrganizationID(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization ID",
				err.Error(),
			).JSON(e)
		}

		response, err := buildScoreboardResponse(e.App, orgID, true)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"scoreboard",
				"failed to build scoreboard response",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, response)
	}
}

// HandleAllResults returns results for all entities
func HandleAllResults() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		response, err := buildScoreboardResponse(e.App, "", false)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"scoreboard",
				"failed to build scoreboard response",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, response)
	}
}

// buildScoreboardResponse aggregates data and builds the response
func buildScoreboardResponse(app core.App, ownerID string, userSpecific bool) (*ScoreboardResponse, error) {
	response := &ScoreboardResponse{}

	// Placeholder: Build summary data
	wallets, err := aggregateWalletResults(app, ownerID, userSpecific)
	if err != nil {
		return nil, err
	}
	response.Summary.Wallets = wallets

	issuers, err := aggregateIssuerResults(app, ownerID, userSpecific)
	if err != nil {
		return nil, err
	}
	response.Summary.Issuers = issuers

	verifiers, err := aggregateVerifierResults(app, ownerID, userSpecific)
	if err != nil {
		return nil, err
	}
	response.Summary.Verifiers = verifiers

	pipelines, err := aggregatePipelineResults(app, ownerID, userSpecific)
	if err != nil {
		return nil, err
	}
	response.Summary.Pipelines = pipelines

	// Build OpenTelemetry formatted data
	otelData := buildOTelData(wallets, issuers, verifiers, pipelines)
	response.OTelData = otelData

	return response, nil
}

// aggregateWalletResults aggregates wallet test results
func aggregateWalletResults(app core.App, ownerID string, userSpecific bool) ([]ScoreboardEntry, error) {
	// TODO: Implement real database queries
	// - Query wallets collection filtered by ownerID if userSpecific=true
	// - Join with wallet_actions and pipeline_results to get test run data
	// - Calculate success/failure counts and rates from actual test results
	// - Group by wallet and aggregate metrics

	// Placeholder: For now, returning example data
	// In production, this would query the database based on ownerID and userSpecific parameters
	entries := []ScoreboardEntry{
		{
			ID:           "wallet_1",
			Name:         "Example Wallet",
			Type:         "wallet",
			TotalRuns:    10,
			SuccessCount: 8,
			FailureCount: 2,
			SuccessRate:  80.0,
			LastRun:      time.Now().Format(time.RFC3339),
		},
	}
	return entries, nil
}

// aggregateIssuerResults aggregates credential issuer test results
func aggregateIssuerResults(app core.App, ownerID string, userSpecific bool) ([]ScoreboardEntry, error) {
	// TODO: Implement real database queries
	// - Query credential_issuers collection filtered by ownerID if userSpecific=true
	// - Join with pipeline_results to get test run data
	// - Calculate success/failure counts and rates from actual test results
	// - Group by issuer and aggregate metrics

	// Placeholder: For now, returning example data
	entries := []ScoreboardEntry{
		{
			ID:           "issuer_1",
			Name:         "Example Issuer",
			Type:         "issuer",
			TotalRuns:    15,
			SuccessCount: 14,
			FailureCount: 1,
			SuccessRate:  93.33,
			LastRun:      time.Now().Format(time.RFC3339),
		},
	}
	return entries, nil
}

// aggregateVerifierResults aggregates verifier test results
func aggregateVerifierResults(app core.App, ownerID string, userSpecific bool) ([]ScoreboardEntry, error) {
	// TODO: Implement real database queries
	// - Query verifiers collection filtered by ownerID if userSpecific=true
	// - Join with use_cases_verifications and pipeline_results to get test run data
	// - Calculate success/failure counts and rates from actual test results
	// - Group by verifier and aggregate metrics

	// Placeholder: For now, returning example data
	entries := []ScoreboardEntry{
		{
			ID:           "verifier_1",
			Name:         "Example Verifier",
			Type:         "verifier",
			TotalRuns:    12,
			SuccessCount: 11,
			FailureCount: 1,
			SuccessRate:  91.67,
			LastRun:      time.Now().Format(time.RFC3339),
		},
	}
	return entries, nil
}

// aggregatePipelineResults aggregates pipeline test results
func aggregatePipelineResults(app core.App, ownerID string, userSpecific bool) ([]ScoreboardEntry, error) {
	// TODO: Implement real database queries
	// - Query pipelines collection filtered by ownerID if userSpecific=true
	// - Join with pipeline_results to get test run data
	// - Calculate success/failure counts and rates from actual test results
	// - Group by pipeline and aggregate metrics

	// Placeholder: For now, returning example data
	entries := []ScoreboardEntry{
		{
			ID:           "pipeline_1",
			Name:         "Example Pipeline",
			Type:         "pipeline",
			TotalRuns:    20,
			SuccessCount: 18,
			FailureCount: 2,
			SuccessRate:  90.0,
			LastRun:      time.Now().Format(time.RFC3339),
		},
	}
	return entries, nil
}

// buildOTelData converts scoreboard entries to OpenTelemetry format
func buildOTelData(wallets, issuers, verifiers, pipelines []ScoreboardEntry) OTelTracesData {
	var allSpans []OTelSpan
	now := time.Now().UnixNano()

	// Convert each entry to a span
	for _, entry := range wallets {
		allSpans = append(allSpans, entryToSpan(entry, now))
	}
	for _, entry := range issuers {
		allSpans = append(allSpans, entryToSpan(entry, now))
	}
	for _, entry := range verifiers {
		allSpans = append(allSpans, entryToSpan(entry, now))
	}
	for _, entry := range pipelines {
		allSpans = append(allSpans, entryToSpan(entry, now))
	}

	return OTelTracesData{
		ResourceSpans: []OTelResourceSpan{
			{
				Resource: struct {
					Attributes []OTelAttribute `json:"attributes"`
				}{
					Attributes: []OTelAttribute{
						{Key: "service.name", Value: "credimi"},
						{Key: "service.version", Value: "1.0.0"},
					},
				},
				ScopeSpans: []struct {
					Scope struct {
						Name    string `json:"name"`
						Version string `json:"version"`
					} `json:"scope"`
					Spans []OTelSpan `json:"spans"`
				}{
					{
						Scope: struct {
							Name    string `json:"name"`
							Version string `json:"version"`
						}{
							Name:    "credimi.scoreboard",
							Version: "1.0.0",
						},
						Spans: allSpans,
					},
				},
			},
		},
	}
}

// entryToSpan converts a ScoreboardEntry to an OpenTelemetry span
func entryToSpan(entry ScoreboardEntry, now int64) OTelSpan {
	status := "OK"
	if entry.SuccessRate < 100 {
		status = "ERROR"
	}

	// Generate OpenTelemetry-compliant IDs
	// TraceID: 32-character hex string (16 bytes)
	// SpanID: 16-character hex string (8 bytes)
	traceID := generateOTelTraceID()
	spanID := generateOTelSpanID()

	return OTelSpan{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      entry.Name,
		Kind:      "SPAN_KIND_INTERNAL",
		StartTime: now - int64(time.Hour),
		EndTime:   now,
		Attributes: []OTelAttribute{
			{Key: "entity.id", Value: entry.ID},
			{Key: "entity.name", Value: entry.Name},
			{Key: "entity.type", Value: entry.Type},
			{Key: "test.total_runs", Value: entry.TotalRuns},
			{Key: "test.success_count", Value: entry.SuccessCount},
			{Key: "test.failure_count", Value: entry.FailureCount},
			{Key: "test.success_rate", Value: entry.SuccessRate},
			{Key: "test.last_run", Value: entry.LastRun},
		},
		Status: OTelStatus{
			Code:    status,
			Message: "",
		},
	}
}

// generateOTelTraceID generates a 32-character hex string for OpenTelemetry TraceID
func generateOTelTraceID() string {
	b := make([]byte, 16) // 16 bytes = 32 hex characters
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based generation if random fails
		return fmt.Sprintf("%032d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// generateOTelSpanID generates a 16-character hex string for OpenTelemetry SpanID
func generateOTelSpanID() string {
	b := make([]byte, 8) // 8 bytes = 16 hex characters
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based generation if random fails
		return fmt.Sprintf("%016d", time.Now().UnixNano()%10000000000000000)
	}
	return hex.EncodeToString(b)
}
*/
