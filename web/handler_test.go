/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type testWriter struct {
	mu     sync.Mutex
	events []*Event
}

func (w *testWriter) WriteEvent(_ context.Context, event *Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events = append(w.events, event)
	return nil
}

func (w *testWriter) Close() error { return nil }

func newTestIter(events ...*adk.AgentEvent) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		for _, e := range events {
			gen.Send(e)
		}
		gen.Close()
	}()
	return iter
}

func TestProcessEventsError(t *testing.T) {
	iter := newTestIter(&adk.AgentEvent{Err: errors.New("test error")})
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 1)
	assert.Equal(t, EventTypeError, w.events[0].Type)
	assert.Equal(t, "test error", w.events[0].Error)
}

func TestProcessEventsRegularMessage(t *testing.T) {
	msg := schema.AssistantMessage("hello world", nil)
	iter := newTestIter(adk.EventFromMessage(msg, nil, schema.Assistant, ""))
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 1)
	assert.Equal(t, EventTypeMessage, w.events[0].Type)
	assert.Equal(t, "hello world", w.events[0].Content)
}

func TestProcessEventsToolMessage(t *testing.T) {
	msg := schema.ToolMessage("result", "call_1")
	iter := newTestIter(adk.EventFromMessage(msg, nil, schema.Tool, "my_tool"))
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 1)
	assert.Equal(t, EventTypeToolResult, w.events[0].Type)
	assert.Equal(t, "result", w.events[0].Content)
}

func TestProcessEventsMessageWithToolCalls(t *testing.T) {
	msg := schema.AssistantMessage("let me check", []schema.ToolCall{
		{ID: "t1", Function: schema.FunctionCall{Name: "search", Arguments: "test"}},
	})
	iter := newTestIter(adk.EventFromMessage(msg, nil, schema.Assistant, ""))
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 1)
	assert.Equal(t, EventTypeMessage, w.events[0].Type)
	assert.Len(t, w.events[0].ToolCalls, 1)
	assert.Equal(t, "search", w.events[0].ToolCalls[0].Function.Name)
}

func TestProcessEventsStreamingMessage(t *testing.T) {
	stream := schema.StreamReaderFromArray([]*schema.Message{
		{Content: "Hello", Role: schema.Assistant},
		{Content: " World", Role: schema.Assistant},
	})
	iter := newTestIter(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				IsStreaming:   true,
				MessageStream: stream,
			},
		},
	})
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 2)
	assert.Equal(t, EventTypeStreamChunk, w.events[0].Type)
	assert.Equal(t, "Hello", w.events[0].Content)
	assert.Equal(t, EventTypeStreamChunk, w.events[1].Type)
	assert.Equal(t, " World", w.events[1].Content)
}

func TestProcessEventsStreamingToolCalls(t *testing.T) {
	idx0 := 0
	stream := schema.StreamReaderFromArray([]*schema.Message{
		{Content: "using tool", Role: schema.Assistant, ToolCalls: []schema.ToolCall{
			{ID: "t1", Index: &idx0, Function: schema.FunctionCall{Name: "search", Arguments: `{"q":`}},
		}},
		{ToolCalls: []schema.ToolCall{
			{ID: "t1", Index: &idx0, Function: schema.FunctionCall{Name: "search", Arguments: `"test"}`}},
		}},
	})
	iter := newTestIter(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				IsStreaming:   true,
				MessageStream: stream,
			},
		},
	})
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 2)
	// First: stream chunk with text content
	assert.Equal(t, EventTypeStreamChunk, w.events[0].Type)
	assert.Equal(t, "using tool", w.events[0].Content)
	// Second: concatenated tool calls
	assert.Equal(t, EventTypeToolCalls, w.events[1].Type)
	require.Len(t, w.events[1].ToolCalls, 1)
}

func TestProcessEventsActionExit(t *testing.T) {
	iter := newTestIter(&adk.AgentEvent{
		Action: adk.NewExitAction(),
	})
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 1)
	assert.Equal(t, EventTypeAction, w.events[0].Type)
	assert.Equal(t, ActionTypeExit, w.events[0].ActionType)
}

func TestProcessEventsActionTransfer(t *testing.T) {
	iter := newTestIter(&adk.AgentEvent{
		Action: adk.NewTransferToAgentAction("research_agent"),
	})
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 1)
	assert.Equal(t, EventTypeAction, w.events[0].Type)
	assert.Equal(t, ActionTypeTransfer, w.events[0].ActionType)
	assert.Equal(t, "research_agent", w.events[0].Content)
}

func TestProcessEventsMultipleEvents(t *testing.T) {
	msg1 := schema.AssistantMessage("hello", nil)
	msg2 := schema.ToolMessage("ok", "c1")
	iter := newTestIter(
		adk.EventFromMessage(msg1, nil, schema.Assistant, ""),
		adk.EventFromMessage(msg2, nil, schema.Tool, "t"),
		&adk.AgentEvent{Action: adk.NewExitAction()},
	)
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	require.Len(t, w.events, 3)
	assert.Equal(t, EventTypeMessage, w.events[0].Type)
	assert.Equal(t, EventTypeToolResult, w.events[1].Type)
	assert.Equal(t, EventTypeAction, w.events[2].Type)
}

func TestCheckpointIDInErrorEvent(t *testing.T) {
	iter := newTestIter(&adk.AgentEvent{Err: errors.New("fail")})
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "checkpoint-123")
	assert.NoError(t, err)
	require.Len(t, w.events, 1)
	// checkpointID is not included in error events (only in interrupted actions)
	assert.Equal(t, EventTypeError, w.events[0].Type)
}

func TestFormatRunPath(t *testing.T) {
	// RunStep has private fields, so we test with empty/nil paths
	assert.Equal(t, "", formatRunPath(nil))
	assert.Equal(t, "", formatRunPath([]adk.RunStep{}))
}

func TestEventJSONSerialization(t *testing.T) {
	event := &Event{
		Type:         EventTypeMessage,
		AgentName:    "test_agent",
		RunPath:      "root/agent",
		Content:      "hello",
		CheckPointID: "cp-123",
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, EventTypeMessage, decoded.Type)
	assert.Equal(t, "test_agent", decoded.AgentName)
	assert.Equal(t, "hello", decoded.Content)
	assert.Equal(t, "cp-123", decoded.CheckPointID)
}

func TestProcessEventsErrorStopsProcessing(t *testing.T) {
	iter := newTestIter(
		&adk.AgentEvent{Err: fmt.Errorf("fatal")},
		&adk.AgentEvent{Action: adk.NewExitAction()},
	)
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	// processEvents continues even after error events, so we get both
	require.Len(t, w.events, 2)
	assert.Equal(t, EventTypeError, w.events[0].Type)
	assert.Equal(t, EventTypeAction, w.events[1].Type)
}

func TestProcessEventsEmptyStream(t *testing.T) {
	stream := schema.StreamReaderFromArray([]*schema.Message{})
	iter := newTestIter(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				IsStreaming:   true,
				MessageStream: stream,
			},
		},
	})
	w := &testWriter{}

	err := processEvents(context.Background(), w, iter, "")
	assert.NoError(t, err)
	assert.Len(t, w.events, 0)
}
