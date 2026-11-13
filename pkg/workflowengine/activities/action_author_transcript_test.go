// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	updatepb "go.temporal.io/api/update/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var transcriptStart = time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)

// chatHistory builds chat history events, one second apart, with payloads
// encoded as Credimi's data converter encodes them.
type chatHistory struct {
	t      *testing.T
	events []*historypb.HistoryEvent
}

func (h *chatHistory) add(eventType enumspb.EventType, event *historypb.HistoryEvent) int64 {
	id := int64(len(h.events) + 1)
	event.EventId = id
	event.EventType = eventType
	event.EventTime = timestamppb.New(transcriptStart.Add(time.Duration(id) * time.Second))
	h.events = append(h.events, event)
	return id
}

func (h *chatHistory) payloads(value any) *commonpb.Payloads {
	h.t.Helper()
	payloads, err := temporalcrypto.DataConverter().ToPayloads(value)
	require.NoError(h.t, err)
	return payloads
}

func (h *chatHistory) update(name string, args *commonpb.Payloads) {
	h.add(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED, &historypb.HistoryEvent{
		Attributes: &historypb.HistoryEvent_WorkflowExecutionUpdateAcceptedEventAttributes{
			WorkflowExecutionUpdateAcceptedEventAttributes: &historypb.WorkflowExecutionUpdateAcceptedEventAttributes{
				AcceptedRequest: &updatepb.Request{
					Input: &updatepb.Input{Name: name, Args: args},
				},
			},
		},
	})
}

func (h *chatHistory) activity(name string, input, result *commonpb.Payloads) {
	scheduled := h.add(enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED, &historypb.HistoryEvent{
		Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
			ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
				ActivityType: &commonpb.ActivityType{Name: name},
				Input:        input,
			},
		},
	})
	h.add(enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED, &historypb.HistoryEvent{
		Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
			ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
				ScheduledEventId: scheduled,
				Result:           result,
			},
		},
	})
}

func (h *chatHistory) ask(text string) {
	h.update(actionAuthorMessageUpdate, h.payloads(map[string]any{
		"type":    "ask",
		"payload": map[string]any{"text": text},
	}))
}

func (h *chatHistory) model(output ...map[string]any) {
	h.activity(
		actionAuthorModelActivity,
		h.payloads(map[string]any{"input": "ignored"}),
		h.payloads(map[string]any{"output": output, "usage": map[string]any{"requests": 1}}),
	)
}

func (h *chatHistory) iterator() *fcafHistoryIterator {
	return &fcafHistoryIterator{events: h.events}
}

func transcriptTime(second int) string {
	return transcriptStart.Add(time.Duration(second) * time.Second).Format(time.RFC3339)
}

func TestParseActionAuthorTranscript(t *testing.T) {
	setFCAFTestEnv(t)
	h := &chatHistory{t: t}
	h.ask("Write the onboarding")
	h.update(actionAuthorMessageUpdate, h.payloads(map[string]any{"type": "close"}))
	h.model(
		map[string]any{"type": "reasoning", "summary": []any{}},
		map[string]any{"type": "message", "role": "assistant", "content": []any{
			map[string]any{"type": "output_text", "text": "Let me look at the screen."},
		}},
		map[string]any{
			"type":      "function_call",
			"call_id":   "call-1",
			"name":      "maestro_inspect_screen",
			"arguments": `{"depth":2}`,
		},
		map[string]any{"type": "message", "content": []any{
			map[string]any{"type": "output_text", "text": "  "},
		}},
	)
	// A reference to an offloaded payload.
	h.activity(actionAuthorModelActivity, nil, &commonpb.Payloads{Payloads: []*commonpb.Payload{{
		Metadata: map[string][]byte{
			"encoding":    []byte("json/protobuf"),
			"messageType": []byte(externalStorageMessage),
		},
		Data: []byte(`{"driverName":"local-file","claimData":{"key":"abc.bin"}}`),
		ExternalPayloads: []*commonpb.Payload_ExternalPayloadDetails{
			{SizeBytes: 2_000_000},
		},
	}}})
	// An undecodable user message.
	h.update(actionAuthorMessageUpdate, &commonpb.Payloads{Payloads: []*commonpb.Payload{{
		Metadata: map[string][]byte{"encoding": []byte("json/plain")},
		Data:     []byte("{not json"),
	}}})
	h.update(actionAuthorCallbackUpdate, h.payloads(map[string]any{
		"tool_id": "ask-user-1",
		"result":  "yes",
	}))
	h.update(actionAuthorCallbackUpdate, h.payloads(map[string]any{
		"tool_id": "save-review-1",
		"result":  map[string]any{"accepted": true, "approval": "signed"},
		"error":   nil,
	}))
	h.activity(
		SaveAuthoredWalletActionActivityName,
		h.payloads(map[string]any{
			"config": map[string]any{},
			"payload": map[string]any{
				"name": "onboarding",
				"code": "appId: eu.wallet\n---\n- launchApp\n",
				"plan": map[string]any{"slot": "onboarding"},
			},
		}),
		h.payloads(map[string]any{"output": map[string]any{
			"record_id": "rec1",
			"created":   false,
			"replaced":  true,
		}}),
	)
	h.activity("Run a mobile test flow", h.payloads(map[string]any{}), h.payloads(map[string]any{}))
	h.model(map[string]any{"type": "message", "content": []any{
		map[string]any{"type": "output_text", "text": "Saved."},
	}})

	entries, err := ParseActionAuthorTranscript(h.iterator())
	require.NoError(t, err)
	accepted := true
	require.Equal(t, []TranscriptEntry{
		{Kind: TranscriptUser, Time: transcriptTime(1), Text: "Write the onboarding"},
		{Kind: TranscriptAssistant, Time: transcriptTime(4), Text: "Let me look at the screen."},
		{
			Kind:      TranscriptToolCall,
			Time:      transcriptTime(4),
			Tool:      "maestro_inspect_screen",
			Arguments: "{\n  \"depth\": 2\n}",
		},
		{Kind: TranscriptUnavailable, Time: transcriptTime(6)},
		{Kind: TranscriptUnavailable, Time: transcriptTime(7)},
		{Kind: TranscriptReview, Time: transcriptTime(9), Accepted: &accepted},
		{
			Kind:     TranscriptSaved,
			Time:     transcriptTime(11),
			Name:     "onboarding",
			Code:     "appId: eu.wallet\n---\n- launchApp\n",
			Replaced: true,
		},
		{Kind: TranscriptAssistant, Time: transcriptTime(15), Text: "Saved."},
	}, entries)
}

func TestParseActionAuthorTranscriptCapsEntries(t *testing.T) {
	setFCAFTestEnv(t)
	h := &chatHistory{t: t}
	for range TranscriptMaxEntries + 5 {
		h.ask("again")
	}
	entries, err := ParseActionAuthorTranscript(h.iterator())
	require.NoError(t, err)
	require.Len(t, entries, TranscriptMaxEntries+1)
	require.Equal(t, TranscriptUser, entries[TranscriptMaxEntries-1].Kind)
	require.Equal(t, TranscriptUnavailable, entries[TranscriptMaxEntries].Kind)
}

func TestTranscriptArguments(t *testing.T) {
	require.Equal(t, "not json", transcriptArguments("not json"))
	long := transcriptArguments(`"` + strings.Repeat("é", 5000) + `"`)
	require.Len(t, []rune(long), TranscriptMaxArguments)
	require.True(t, strings.HasSuffix(long, "é…"))
}

func TestReadActionAuthorTranscriptReportsAMissingChat(t *testing.T) {
	setFCAFTestEnv(t)
	c := &temporalmocks.Client{}
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		"chat-author",
		"",
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(&failingHistoryIterator{err: serviceerror.NewNotFound("no chat")})
	_, err := ReadActionAuthorTranscript(t.Context(), c, "chat-author")
	require.ErrorIs(t, err, ErrActionAuthorTranscriptNotFound)
}
