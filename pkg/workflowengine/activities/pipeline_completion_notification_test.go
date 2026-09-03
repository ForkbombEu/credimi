// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestSendPipelineCompletionNotificationActivityName(t *testing.T) {
	require.Equal(
		t,
		"Send pipeline completion notification",
		NewSendPipelineCompletionNotificationActivity().Name(),
	)
}

func TestSendPipelineCompletionNotificationActivitySuccess(t *testing.T) {
	var mu sync.Mutex
	var gotPath, gotAPIKey, gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("Credimi-Api-Key")
		gotContentType = r.Header.Get("Content-Type")
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "internal-admin-key")

	activity := NewSendPipelineCompletionNotificationActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: SendPipelineCompletionNotificationInput{
			AppURL:     server.URL,
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Result:     "success",
		},
	})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, "/api/web-push/pipeline-completed", gotPath)
	require.Equal(t, "internal-admin-key", gotAPIKey)
	require.Equal(t, workflowengine.MIMEApplicationJSON, gotContentType)

	var body map[string]string
	require.NoError(t, json.Unmarshal([]byte(gotBody), &body))
	require.Equal(t, server.URL, body["app_url"])
	require.Equal(t, "wf-1", body["workflow_id"])
	require.Equal(t, "run-1", body["run_id"])
	require.Equal(t, "success", body["result"])
}

func TestSendPipelineCompletionNotificationActivityMissingFields(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "internal-admin-key")

	activity := NewSendPipelineCompletionNotificationActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: SendPipelineCompletionNotificationInput{
			WorkflowID: "wf-1",
			RunID:      "run-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

func TestSendPipelineCompletionNotificationActivityMissingAdminKey(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "")

	activity := NewSendPipelineCompletionNotificationActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: SendPipelineCompletionNotificationInput{
			AppURL:     "http://127.0.0.1:1",
			WorkflowID: "wf-1",
			RunID:      "run-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CREDIMI_INTERNAL_ADMIN_KEY is required")
}

func TestSendPipelineCompletionNotificationActivityNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "internal-admin-key")

	activity := NewSendPipelineCompletionNotificationActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: SendPipelineCompletionNotificationInput{
			AppURL:     server.URL,
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Result:     "failed",
		},
	})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "pipeline completion notification status"))
}
