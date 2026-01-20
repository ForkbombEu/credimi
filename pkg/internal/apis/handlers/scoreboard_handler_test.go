// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"
)

func TestScoreboardEntryStructure(t *testing.T) {
	entry := ScoreboardEntry{
		ID:           "test_id",
		Name:         "Test Entry",
		Type:         "wallet",
		TotalRuns:    10,
		SuccessCount: 8,
		FailureCount: 2,
		SuccessRate:  80.0,
		LastRun:      "2025-01-12T00:00:00Z",
	}

	if entry.ID != "test_id" {
		t.Errorf("Expected ID to be 'test_id', got '%s'", entry.ID)
	}
	if entry.SuccessRate != 80.0 {
		t.Errorf("Expected SuccessRate to be 80.0, got %f", entry.SuccessRate)
	}
}

func TestOTelSpanStructure(t *testing.T) {
	span := OTelSpan{
		TraceID:   "trace123",
		SpanID:    "span456",
		Name:      "Test Span",
		Kind:      "SPAN_KIND_INTERNAL",
		StartTime: 1000000000,
		EndTime:   2000000000,
		Attributes: []OTelAttribute{
			{Key: "test.key", Value: "test.value"},
		},
		Status: OTelStatus{
			Code:    "OK",
			Message: "",
		},
	}

	if span.TraceID != "trace123" {
		t.Errorf("Expected TraceID to be 'trace123', got '%s'", span.TraceID)
	}
	if span.Status.Code != "OK" {
		t.Errorf("Expected Status.Code to be 'OK', got '%s'", span.Status.Code)
	}
}

func TestEntryToSpan(t *testing.T) {
	entry := ScoreboardEntry{
		ID:           "test_wallet",
		Name:         "Test Wallet",
		Type:         "wallet",
		TotalRuns:    5,
		SuccessCount: 4,
		FailureCount: 1,
		SuccessRate:  80.0,
		LastRun:      "2025-01-12T00:00:00Z",
	}

	span := entryToSpan(entry, 1000000000)

	// TraceID should be 32 hex characters
	if len(span.TraceID) != 32 {
		t.Errorf("Expected TraceID to be 32 characters, got %d", len(span.TraceID))
	}
	
	// SpanID should be 16 hex characters
	if len(span.SpanID) != 16 {
		t.Errorf("Expected SpanID to be 16 characters, got %d", len(span.SpanID))
	}
	
	if span.Name != "Test Wallet" {
		t.Errorf("Expected Name to be 'Test Wallet', got '%s'", span.Name)
	}

	// Check attributes
	foundEntityID := false
	foundSuccessRate := false
	for _, attr := range span.Attributes {
		if attr.Key == "entity.id" && attr.Value == "test_wallet" {
			foundEntityID = true
		}
		if attr.Key == "test.success_rate" && attr.Value == 80.0 {
			foundSuccessRate = true
		}
	}

	if !foundEntityID {
		t.Error("Expected to find entity.id attribute")
	}
	if !foundSuccessRate {
		t.Error("Expected to find test.success_rate attribute")
	}

	// Status should be ERROR for non-100% success rate
	if span.Status.Code != "ERROR" {
		t.Errorf("Expected Status.Code to be 'ERROR' for 80%% success rate, got '%s'", span.Status.Code)
	}
}

func TestBuildOTelData(t *testing.T) {
	wallets := []ScoreboardEntry{
		{
			ID:           "wallet1",
			Name:         "Wallet 1",
			Type:         "wallet",
			TotalRuns:    10,
			SuccessCount: 10,
			FailureCount: 0,
			SuccessRate:  100.0,
			LastRun:      "2025-01-12T00:00:00Z",
		},
	}

	issuers := []ScoreboardEntry{}
	verifiers := []ScoreboardEntry{}
	pipelines := []ScoreboardEntry{}

	otelData := buildOTelData(wallets, issuers, verifiers, pipelines)

	if len(otelData.ResourceSpans) == 0 {
		t.Error("Expected at least one ResourceSpan")
	}

	if len(otelData.ResourceSpans) > 0 {
		rs := otelData.ResourceSpans[0]
		if len(rs.Resource.Attributes) == 0 {
			t.Error("Expected resource attributes to be set")
		}

		if len(rs.ScopeSpans) == 0 {
			t.Error("Expected at least one ScopeSpan")
		}

		if len(rs.ScopeSpans) > 0 {
			ss := rs.ScopeSpans[0]
			if ss.Scope.Name != "credimi.scoreboard" {
				t.Errorf("Expected scope name to be 'credimi.scoreboard', got '%s'", ss.Scope.Name)
			}

			if len(ss.Spans) != 1 {
				t.Errorf("Expected exactly 1 span, got %d", len(ss.Spans))
			}
		}
	}
}
