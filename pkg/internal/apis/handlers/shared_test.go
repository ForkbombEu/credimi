// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestDecodeFromTemporalPayload(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(`"hello"`))
	require.Equal(t, "hello", DecodeFromTemporalPayload(encoded))
	require.Equal(t, "not-base64", DecodeFromTemporalPayload("not-base64"))
	require.Equal(t, "", DecodeFromTemporalPayload(""))
}

func TestCalculateAndFormatDuration(t *testing.T) {
	require.Equal(t, "", calculateDuration("", ""))
	require.Equal(t, "", calculateDuration("invalid", "2025-01-01T00:00:00Z"))

	start := "2025-01-01T00:00:00Z"
	end := "2025-01-01T00:00:01.500Z"
	require.Equal(t, "1s", calculateDuration(start, end))

	require.Equal(t, "", formatDuration(-1*time.Second))
	require.Equal(t, "1s", formatDuration(1500*time.Millisecond))
}

func TestFetchWorkflowFailure(t *testing.T) {
	t.Run("no events", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On(
				"GetWorkflowHistory",
				mock.Anything,
				"wf-1",
				"run-1",
				false,
				enums.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
			).
			Return(&fakeHistoryIterator{}, nil).
			Once()

		result := fetchWorkflowFailure(context.Background(), mockClient, "wf-1", "run-1")
		require.Nil(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("close event without failure cause", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On(
				"GetWorkflowHistory",
				mock.Anything,
				"wf-2",
				"run-2",
				false,
				enums.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
			).
			Return(&fakeHistoryIterator{
				events: []*historypb.HistoryEvent{
					{
						Attributes: &historypb.HistoryEvent_WorkflowExecutionFailedEventAttributes{
							WorkflowExecutionFailedEventAttributes: &historypb.WorkflowExecutionFailedEventAttributes{
								Failure: &failure.Failure{},
							},
						},
					},
				},
			}, nil).
			Once()

		result := fetchWorkflowFailure(context.Background(), mockClient, "wf-2", "run-2")
		require.Nil(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("close event with failure cause", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On(
				"GetWorkflowHistory",
				mock.Anything,
				"wf-3",
				"run-3",
				false,
				enums.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
			).
			Return(&fakeHistoryIterator{
				events: []*historypb.HistoryEvent{
					{
						Attributes: &historypb.HistoryEvent_WorkflowExecutionFailedEventAttributes{
							WorkflowExecutionFailedEventAttributes: &historypb.WorkflowExecutionFailedEventAttributes{
								Failure: &failure.Failure{
									Cause: &failure.Failure{Message: "boom"},
								},
							},
						},
					},
				},
			}, nil).
			Once()

		result := fetchWorkflowFailure(context.Background(), mockClient, "wf-3", "run-3")
		require.NotNil(t, result)
		require.Equal(t, "boom", *result)
		mockClient.AssertExpectations(t)
	})
}

func TestGetStringFromMap(t *testing.T) {
	require.Equal(t, "", getStringFromMap(nil, "key"))
	require.Equal(t, "", getStringFromMap(map[string]any{"key": 123}, "key"))
	require.Equal(t, "value", getStringFromMap(map[string]any{"key": "value"}, "key"))
}
