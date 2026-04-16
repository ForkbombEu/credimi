// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
)

func TestExtractPipelineOutput(t *testing.T) {
	dataCtx := map[string]any{
		"inputs": map[string]any{"user_id": 123},
		"step1": map[string]any{
			"outputs": map[string]any{"id": 1, "name": "Mario"},
		},
		"step2": map[string]any{
			"outputs": map[string]any{"total": 100},
		},
	}

	result := ExtractPipelineOutput(dataCtx)

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
	if result["step1"].(map[string]any)["id"] != 1 {
		t.Error("step1.id should be 1")
	}
	if result["step2"].(map[string]any)["total"] != 100 {
		t.Error("step2.total should be 100")
	}
}

func TestEnrichActivityPayload_HTTP(t *testing.T) {
	payload := &activities.HTTPActivityPayload{
		Method: "POST",
		URL:    "https://api.example.com",
		Body:   map[string]any{"original": "data"},
	}
	pipelineOutput := map[string]any{
		"step1": map[string]any{"id": 123, "name": "Name"},
	}

	result, err := EnrichActivityPayload(payload, "http-request", pipelineOutput)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	httpResult := result.(*activities.HTTPActivityPayload)
	newBody := httpResult.Body.(map[string]any)

	if newBody["step1"] == nil {
		t.Error("Expected step1 in body, got nil")
	}
	if httpResult.Method != "POST" {
		t.Error("Method should remain POST")
	}
}

func TestEnrichActivityPayload_Email(t *testing.T) {
	payload := &activities.SendMailActivityPayload{
		Recipient: "test@example.com",
		Subject:   "Test",
		Body:      "Original body",
	}
	pipelineOutput := map[string]any{
		"step1": map[string]any{"id": 123, "name": "Name"},
	}

	result, err := EnrichActivityPayload(payload, "email", pipelineOutput)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	emailResult := result.(*activities.SendMailActivityPayload)
	
	var bodyMap map[string]any
	err = json.Unmarshal([]byte(emailResult.Body), &bodyMap)
	if err != nil {
		t.Errorf("Body should be valid JSON, got error: %v", err)
	}

	if bodyMap["step1"] == nil {
		t.Error("Expected step1 in body")
	}
	if emailResult.Recipient != "test@example.com" {
		t.Error("Recipient should remain unchanged")
	}
}

func TestEnrichActivityPayload_NoEnrichmentForUnknownType(t *testing.T) {
	payload := &activities.HTTPActivityPayload{
		Method: "GET",
	}
	pipelineOutput := map[string]any{"data": "value"}
	result, err := EnrichActivityPayload(payload, "unknown_type", pipelineOutput)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != payload {
		t.Error("Payload should remain unchanged for unknown type")
	}
}
