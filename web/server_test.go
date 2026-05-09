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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// stubAgent implements adk.Agent with a simple echo response.
type stubAgent struct {
	name string
}

func (a *stubAgent) Name(_ context.Context) string      { return a.name }
func (a *stubAgent) Description(_ context.Context) string { return "test stub agent" }
func (a *stubAgent) Run(_ context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	var lastContent string
	if len(input.Messages) > 0 {
		lastContent = input.Messages[len(input.Messages)-1].Content
	}
	response := "echo: " + lastContent

	if input.EnableStreaming {
		go func() {
			gen.Send(adk.EventFromMessage(nil,
				schema.StreamReaderFromArray([]*schema.Message{
					{Role: schema.Assistant, Content: response[:len(response)/2]},
					{Role: schema.Assistant, Content: response[len(response)/2:]},
				}), schema.Assistant, ""))
			gen.Send(&adk.AgentEvent{Action: adk.NewExitAction()})
			gen.Close()
		}()
	} else {
		go func() {
			gen.Send(adk.EventFromMessage(schema.AssistantMessage(response, nil), nil, schema.Assistant, ""))
			gen.Send(&adk.AgentEvent{Action: adk.NewExitAction()})
			gen.Close()
		}()
	}
	return iter
}

func TestServerE2E(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})

	srv := NewServer(runner, WithAddr("127.0.0.1:0"), WithEnableUI(false))
	require.NotNil(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(200 * time.Millisecond)

	// We can't easily get the random port, so test via handler directly
	// The Start goroutine will block; cancel the ctx after tests
	cancel()
	select {
	case err := <-errCh:
		// http.ErrServerClosed is expected after graceful shutdown
		if err != nil && err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestServerHealthEndpoint(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithEnableUI(false))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}

func TestServerChatSSE(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithEnableUI(false))

	req := httptest.NewRequest("GET", "/chat?query=hello", nil)
	w := httptest.NewRecorder()
	srv.chatHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	body := w.Body.String()
	assert.Contains(t, body, "data: ")
	assert.Contains(t, body, `"type":"stream_chunk"`)
}

func TestServerChatMissingQuery(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithEnableUI(false))

	req := httptest.NewRequest("GET", "/chat", nil)
	w := httptest.NewRecorder()
	srv.chatHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "query parameter is required")
}

func TestServerUIServed(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithEnableUI(true))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler := srv.uiHandler()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Eino Chat")
}

func TestServerCORS(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithAllowedOrigins("http://localhost:3000"))

	handler := srv.wrapCORS(srv.healthHandler)

	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestServerCORSAllOrigins(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithAllowedOrigins("*"))

	handler := srv.wrapCORS(srv.healthHandler)

	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestServerChatPOST(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithEnableUI(false))

	body := strings.NewReader(`{"query":"hello via post"}`)
	req := httptest.NewRequest("POST", "/chat", body)
	w := httptest.NewRecorder()
	srv.chatHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "data: ")
}

func TestServerResumeWithoutStore(t *testing.T) {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           &stubAgent{name: "test"},
		EnableStreaming: true,
	})
	srv := NewServer(runner, WithAddr(":0"), WithEnableUI(false))

	req := httptest.NewRequest("POST", "/resume/fake-id", nil)
	w := httptest.NewRecorder()
	srv.resumeHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
