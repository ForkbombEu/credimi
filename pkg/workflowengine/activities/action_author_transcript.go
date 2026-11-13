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
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

// Transcript entry kinds.
const (
	TranscriptUser        = "user"
	TranscriptAssistant   = "assistant"
	TranscriptToolCall    = "tool_call"
	TranscriptReview      = "review"
	TranscriptSaved       = "saved"
	TranscriptUnavailable = "unavailable"
)

const (
	// TranscriptMaxEntries caps a transcript; one unavailable entry follows the cut.
	TranscriptMaxEntries = 2000
	// TranscriptMaxArguments caps a tool call's pretty-printed arguments, in characters.
	TranscriptMaxArguments = 4000

	actionAuthorMessageUpdate  = "send_agent_message"
	actionAuthorCallbackUpdate = "provide_callback_result"
	actionAuthorModelActivity  = "invoke_model_activity"
	actionAuthorReviewToolID   = "save-review-"
	// externalStorageEncoding and externalStorageMessage mark the reference a
	// worker's external storage leaves in history instead of a large payload.
	externalStorageEncoding = "json/external-storage-reference"
	externalStorageMessage  = "temporal.api.sdk.v1.ExternalStorageReference"
)

// ErrActionAuthorTranscriptNotFound reports a chat workflow Temporal does not know.
var ErrActionAuthorTranscriptNotFound = errors.New("action author chat not found")

// TranscriptEntry is one item of an action author chat, read from its history.
type TranscriptEntry struct {
	// Kind is user, assistant, tool_call, review, saved or unavailable.
	Kind string `json:"kind"`
	// Time is the RFC 3339 time of the history event.
	Time string `json:"time"`
	Text string `json:"text,omitempty"`
	Tool string `json:"tool,omitempty"`
	// Arguments is the tool call's pretty JSON, at most 4000 characters, cut with "…".
	Arguments string `json:"arguments,omitempty"`
	Accepted  *bool  `json:"accepted,omitempty"`
	Name      string `json:"name,omitempty"`
	Code      string `json:"code,omitempty"`
	Replaced  bool   `json:"replaced,omitempty"`
}

// ReadActionAuthorTranscript reads the transcript of the latest run of the
// chat workflow chatWorkflowID from its history, without querying its worker.
// A chat Temporal does not know returns ErrActionAuthorTranscriptNotFound.
func ReadActionAuthorTranscript(
	ctx context.Context,
	c client.Client,
	chatWorkflowID string,
) ([]TranscriptEntry, error) {
	history := c.GetWorkflowHistory(
		ctx,
		chatWorkflowID,
		"",
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	entries, err := ParseActionAuthorTranscript(history)
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return nil, ErrActionAuthorTranscriptNotFound
	}
	return entries, err
}

// ParseActionAuthorTranscript reads the transcript from the events of a chat
// history, in event order: user messages, the agent's replies and tool
// calls, save review decisions and saved actions. A payload it cannot decode,
// offloaded payloads included, becomes one unavailable entry.
func ParseActionAuthorTranscript(history client.HistoryEventIterator) ([]TranscriptEntry, error) {
	t := transcriptReader{
		entries: []TranscriptEntry{},
		models:  map[int64]bool{},
		saves:   map[int64]*TranscriptEntry{},
		dc:      temporalcrypto.DataConverter(),
	}
	for history.HasNext() {
		event, err := history.Next()
		if err != nil {
			return nil, fmt.Errorf("read action author chat: %w", err)
		}
		t.apply(event)
		if len(t.entries) >= TranscriptMaxEntries {
			t.entries = append(t.entries[:TranscriptMaxEntries], TranscriptEntry{
				Kind: TranscriptUnavailable,
				Time: t.entries[TranscriptMaxEntries-1].Time,
			})
			break
		}
	}
	return t.entries, nil
}

type transcriptReader struct {
	entries []TranscriptEntry
	// models holds the scheduled event IDs of model calls.
	models map[int64]bool
	// saves holds the saved entry of each scheduled save, nil when its input
	// could not be decoded.
	saves map[int64]*TranscriptEntry
	dc    converter.DataConverter
}

type transcriptModelResponse struct {
	Output []struct {
		Type      string `json:"type"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (t *transcriptReader) apply(event *historypb.HistoryEvent) {
	at := event.GetEventTime().AsTime().UTC().Format(time.RFC3339)
	switch event.GetEventType() {
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED:
		input := event.GetWorkflowExecutionUpdateAcceptedEventAttributes().
			GetAcceptedRequest().GetInput()
		switch input.GetName() {
		case actionAuthorMessageUpdate:
			t.userMessage(at, input.GetArgs())
		case actionAuthorCallbackUpdate:
			t.review(at, input.GetArgs())
		}
	case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
		attrs := event.GetActivityTaskScheduledEventAttributes()
		switch attrs.GetActivityType().GetName() {
		case actionAuthorModelActivity:
			t.models[event.GetEventId()] = true
		case SaveAuthoredWalletActionActivityName:
			var input struct {
				Payload struct {
					Name string `json:"name"`
					Code string `json:"code"`
				} `json:"payload"`
			}
			if t.decode(attrs.GetInput(), &input) {
				t.saves[event.GetEventId()] = &TranscriptEntry{
					Kind: TranscriptSaved,
					Name: input.Payload.Name,
					Code: input.Payload.Code,
				}
			} else {
				t.saves[event.GetEventId()] = nil
			}
		}
	case enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		attrs := event.GetActivityTaskCompletedEventAttributes()
		scheduled := attrs.GetScheduledEventId()
		if t.models[scheduled] {
			t.modelResponse(at, attrs.GetResult())
			return
		}
		saved, ok := t.saves[scheduled]
		if !ok {
			return
		}
		var result struct {
			Output struct {
				Replaced bool `json:"replaced"`
			} `json:"output"`
		}
		if saved == nil || !t.decode(attrs.GetResult(), &result) {
			t.unavailable(at)
			return
		}
		saved.Time = at
		saved.Replaced = result.Output.Replaced
		t.entries = append(t.entries, *saved)
	}
}

func (t *transcriptReader) userMessage(at string, args *commonpb.Payloads) {
	var message struct {
		Type    string `json:"type"`
		Payload struct {
			Text string `json:"text"`
		} `json:"payload"`
	}
	if !t.decode(args, &message) {
		t.unavailable(at)
		return
	}
	if message.Type != "ask" {
		return
	}
	t.entries = append(t.entries, TranscriptEntry{
		Kind: TranscriptUser,
		Time: at,
		Text: message.Payload.Text,
	})
}

func (t *transcriptReader) review(at string, args *commonpb.Payloads) {
	// Other callbacks carry results of other shapes.
	var callback struct {
		ToolID string          `json:"tool_id"`
		Result json.RawMessage `json:"result"`
	}
	if !t.decode(args, &callback) {
		t.unavailable(at)
		return
	}
	if !strings.HasPrefix(callback.ToolID, actionAuthorReviewToolID) {
		return
	}
	var decision struct {
		Accepted bool `json:"accepted"`
	}
	if json.Unmarshal(callback.Result, &decision) != nil {
		t.unavailable(at)
		return
	}
	t.entries = append(t.entries, TranscriptEntry{
		Kind:     TranscriptReview,
		Time:     at,
		Accepted: &decision.Accepted,
	})
}

func (t *transcriptReader) modelResponse(at string, result *commonpb.Payloads) {
	var response transcriptModelResponse
	if !t.decode(result, &response) {
		t.unavailable(at)
		return
	}
	for _, item := range response.Output {
		switch item.Type {
		case "message":
			texts := []string{}
			for _, content := range item.Content {
				if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
					texts = append(texts, content.Text)
				}
			}
			if len(texts) > 0 {
				t.entries = append(t.entries, TranscriptEntry{
					Kind: TranscriptAssistant,
					Time: at,
					Text: strings.Join(texts, "\n\n"),
				})
			}
		case "function_call":
			t.entries = append(t.entries, TranscriptEntry{
				Kind:      TranscriptToolCall,
				Time:      at,
				Tool:      item.Name,
				Arguments: transcriptArguments(item.Arguments),
			})
		}
	}
}

func (t *transcriptReader) unavailable(at string) {
	t.entries = append(t.entries, TranscriptEntry{Kind: TranscriptUnavailable, Time: at})
}

// decode decodes the first payload of payloads into value, and fails on an
// offloaded payload.
func (t *transcriptReader) decode(payloads *commonpb.Payloads, value any) bool {
	all := payloads.GetPayloads()
	if len(all) == 0 || offloaded(all[0]) {
		return false
	}
	return t.dc.FromPayload(all[0], value) == nil
}

// offloaded reports a reference to a payload kept in a worker's external storage.
func offloaded(payload *commonpb.Payload) bool {
	metadata := payload.GetMetadata()
	return len(payload.GetExternalPayloads()) > 0 ||
		string(metadata[converter.MetadataEncoding]) == externalStorageEncoding ||
		string(metadata[converter.MetadataMessageType]) == externalStorageMessage
}

// transcriptArguments pretty-prints a tool call's JSON arguments, kept as
// they are when they are not JSON, and cuts them to TranscriptMaxArguments
// characters.
func transcriptArguments(arguments string) string {
	var pretty bytes.Buffer
	if json.Indent(&pretty, []byte(arguments), "", "  ") == nil {
		arguments = pretty.String()
	}
	runes := []rune(arguments)
	if len(runes) <= TranscriptMaxArguments {
		return arguments
	}
	return string(runes[:TranscriptMaxArguments-1]) + "…"
}
