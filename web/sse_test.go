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
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEWriterHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := newSSEWriter(w)
	require.NoError(t, err)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	err = sw.Close()
	assert.NoError(t, err)
}

func TestSSEWriterWriteEvent(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := newSSEWriter(w)
	require.NoError(t, err)

	err = sw.WriteEvent(context.Background(), &Event{
		Type:    EventTypeMessage,
		Content: "hello",
	})
	require.NoError(t, err)

	body := w.Body.String()
	assert.Contains(t, body, "data: ")
	assert.Contains(t, body, `"type":"message"`)
	assert.Contains(t, body, `"content":"hello"`)
}

func TestSSEWriterMultipleEvents(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := newSSEWriter(w)
	require.NoError(t, err)

	err = sw.WriteEvent(context.Background(), &Event{Type: EventTypeMessage, Content: "a"})
	require.NoError(t, err)
	err = sw.WriteEvent(context.Background(), &Event{Type: EventTypeStreamChunk, Content: "b"})
	require.NoError(t, err)

	body := w.Body.String()
	events := strings.Split(strings.TrimSpace(body), "\n\n")
	assert.Len(t, events, 2)
	for _, eventStr := range events {
		assert.True(t, strings.HasPrefix(strings.TrimSpace(eventStr), "data: "), "expected SSE data: prefix, got: %s", eventStr)
	}
}

func TestSSEWriterWriteEventWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	// Use http.ResponseWriter proxy that checks for ctx cancellation
	w := httptest.NewRecorder()
	sw, err := newSSEWriter(w)
	require.NoError(t, err)

	// sseWriter checks ctx.Done() and returns context.Canceled
	err = sw.WriteEvent(ctx, &Event{Type: EventTypeMessage})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
}

func TestSSEWriterFlusherError(t *testing.T) {
	bw := &nonFlusherWriter{HeaderMap: http.Header{}}
	_, err := newSSEWriter(bw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "streaming not supported")
}

// nonFlusherWriter is an http.ResponseWriter that does not implement http.Flusher.
type nonFlusherWriter struct {
	HeaderMap http.Header
}

func (w *nonFlusherWriter) Header() http.Header         { return w.HeaderMap }
func (w *nonFlusherWriter) Write([]byte) (int, error)    { return 0, nil }
func (w *nonFlusherWriter) WriteHeader(int)              {}

func TestSSEEventFormat(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := newSSEWriter(w)
	require.NoError(t, err)

	err = sw.WriteEvent(context.Background(), &Event{
		Type:         EventTypeAction,
		AgentName:    "agent1",
		ActionType:   ActionTypeInterrupted,
		CheckPointID: "cp-123",
	})
	require.NoError(t, err)

	scanner := bufio.NewScanner(w.Body)
	var sseLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			sseLine = line
			break
		}
	}

	assert.True(t, strings.HasPrefix(sseLine, "data: "))
	// The JSON should contain the checkpoint ID
	assert.Contains(t, sseLine, `"checkpoint_id":"cp-123"`)
}
