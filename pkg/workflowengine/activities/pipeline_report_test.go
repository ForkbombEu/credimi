// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func reportTestHistory(t *testing.T) []*historypb.HistoryEvent {
	t.Helper()
	def := &pipelineinternal.WorkflowDefinition{
		Name: "report-pipeline",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "credential-step", Use: "credential-offer"}},
		},
	}
	return []*historypb.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
					Input: fcafPayloads(t, map[string]any{"workflow_definition": def}),
				},
			},
		},
		{
			EventId:   5,
			EventType: enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED,
			Attributes: &historypb.HistoryEvent_StartChildWorkflowExecutionInitiatedEventAttributes{
				StartChildWorkflowExecutionInitiatedEventAttributes: &historypb.StartChildWorkflowExecutionInitiatedEventAttributes{
					Input: fcafPayloads(t, workflowengine.WorkflowInput{
						Config: map[string]any{workflowengine.StepIDConfigKey: "credential-step"},
					}),
				},
			},
		},
		{
			EventId:   7,
			EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED,
			Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionCompletedEventAttributes{
				ChildWorkflowExecutionCompletedEventAttributes: &historypb.ChildWorkflowExecutionCompletedEventAttributes{
					InitiatedEventId: 5,
					Result: fcafPayloads(t, workflowengine.WorkflowResult{
						Output: map[string]any{"status": "ok"},
					}),
				},
			},
		},
	}
}

func reportTestEvidence() PipelineEvidenceExtractionOutput {
	return PipelineEvidenceExtractionOutput{
		CredentialOffers: []map[string]any{
			{
				"step_id":       "credential-step",
				"credential_id": "tenant/credential",
				"credential_offer": map[string]any{
					"credential_issuer":            "https://issuer.example",
					"credential_configuration_ids": []string{"pid_sd_jwt"},
					"grants": map[string]any{
						"urn:ietf:params:oauth:grant-type:pre-authorized_code": map[string]any{},
					},
				},
			},
		},
		CredentialWellKnowns: []map[string]any{
			{
				"step_id":       "credential-step",
				"credential_id": "tenant/credential",
				"well_known": map[string]any{
					"credential_configurations_supported": map[string]any{
						"pid_sd_jwt": map[string]any{
							"format": "vc+sd-jwt",
							"proof_types_supported": map[string]any{
								"jwt": map[string]any{
									"proof_signing_alg_values_supported": []string{"ES256"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestPipelineReportGenerationActivityBuildsFromHistoryAndStoresMarkdown(t *testing.T) {
	setFCAFTestEnv(t)
	historyClient := &temporalmocks.Client{}
	historyClient.On(
		"GetWorkflowHistory",
		mock.Anything,
		"workflow-1",
		"run-1",
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(&fcafHistoryIterator{events: reportTestHistory(t)})
	temporalclient.SetClientForTests("tenant", historyClient)

	var stored map[string]string
	store := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/pipeline/pipeline-execution-results/report", r.URL.Path)
		require.Equal(t, "test-key", r.Header.Get("Credimi-Api-Key"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&stored))
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(store.Close)

	act := NewPipelineReportGenerationActivity(fcafTestOutputKind)
	res, err := act.Execute(t.Context(), workflowengine.ActivityInput{
		Payload: PipelineReportGenerationInput{
			Namespace:          "tenant",
			WorkflowID:         "workflow-1",
			RunID:              "run-1",
			AppURL:             store.URL,
			PipelineOutputMeta: map[string]any{"setup_warnings": []string{"warning"}},
			Evidence:           reportTestEvidence(),
		},
	})
	require.NoError(t, err)
	historyClient.AssertExpectations(t)

	out, ok := res.Output.(PipelineReportGenerationOutput)
	require.True(t, ok)
	require.Empty(t, out.Warnings)
	require.Equal(t, "workflow-1.md", out.Filename)
	require.Equal(t, "workflow-1", out.Fixture)

	require.Equal(t, "workflow-1", stored["workflow_id"])
	require.Equal(t, "run-1", stored["run_id"])
	require.Equal(t, "workflow-1.md", stored["filename"])
	require.Contains(t, stored["markdown"], "Credimi Conformance Assessment")
	sum := sha256.Sum256([]byte(stored["markdown"]))
	require.Equal(t, hex.EncodeToString(sum[:]), out.MarkdownSHA256)

	encoded, err := json.Marshal(out)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"markdown"`)
}

func TestPipelineReportGenerationActivityWarnsWhenStorageFails(t *testing.T) {
	setFCAFTestEnv(t)
	historyClient := &temporalmocks.Client{}
	historyClient.On("GetWorkflowHistory", mock.Anything, "workflow-1", "run-1", false, mock.Anything).
		Return(&fcafHistoryIterator{events: reportTestHistory(t)})
	temporalclient.SetClientForTests("tenant", historyClient)
	store := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(store.Close)

	act := NewPipelineReportGenerationActivity(fcafTestOutputKind)
	res, err := act.Execute(t.Context(), workflowengine.ActivityInput{
		Payload: PipelineReportGenerationInput{
			Namespace:  "tenant",
			WorkflowID: "workflow-1",
			RunID:      "run-1",
			AppURL:     store.URL,
			Evidence:   reportTestEvidence(),
		},
	})
	require.NoError(t, err)
	out, ok := res.Output.(PipelineReportGenerationOutput)
	require.True(t, ok)
	require.Len(t, out.Warnings, 1)
	require.Contains(t, out.Warnings[0], "pipeline report storage failed")
	require.Contains(t, out.Warnings[0], "status 404")
}

func TestSanitizeReportFilename(t *testing.T) {
	require.Equal(t, "workflow-1.md", sanitizeReportFilename(" workflow/1 ")+".md")
	require.Equal(t, "pipeline-report", sanitizeReportFilename("///"))
}

func TestPipelineReportGenerationActivityValidation(t *testing.T) {
	act := NewPipelineReportGenerationActivity(fcafTestOutputKind)
	_, err := act.Execute(
		t.Context(),
		workflowengine.ActivityInput{Payload: PipelineReportGenerationInput{}},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "workflow_id and run_id are required")

	raw, err := marshalRaw(nil)
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(raw))
}
