// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

func temporalMemoWithLogsCapability(t *testing.T, logs bool) *commonpb.Memo {
	t.Helper()
	payload, err := converter.GetDefaultDataConverter().ToPayload(
		workflowengine.CredimiCapabilities{Logs: logs},
	)
	require.NoError(t, err)
	return &commonpb.Memo{Fields: map[string]*commonpb.Payload{
		workflowengine.CredimiCapabilitiesMemoKey: payload,
	}}
}

func localMemoWithLogsCapability(t *testing.T, logs bool) *Memo {
	t.Helper()
	data, err := json.Marshal(workflowengine.CredimiCapabilities{Logs: logs})
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(data)
	return &Memo{Fields: map[string]*Payload{
		workflowengine.CredimiCapabilitiesMemoKey: {Data: &encoded},
	}}
}

func TestWorkflowExecutionHasLogs(t *testing.T) {
	require.True(t, workflowExecutionHasLogs(&WorkflowExecution{
		Memo: localMemoWithLogsCapability(t, true),
	}))
	require.False(t, workflowExecutionHasLogs(&WorkflowExecution{
		Memo: localMemoWithLogsCapability(t, false),
	}))
	require.False(t, workflowExecutionHasLogs(&WorkflowExecution{}))

	invalid := "not-base64"
	require.False(t, workflowExecutionHasLogs(&WorkflowExecution{Memo: &Memo{
		Fields: map[string]*Payload{
			workflowengine.CredimiCapabilitiesMemoKey: {Data: &invalid},
		},
	}}))
}
