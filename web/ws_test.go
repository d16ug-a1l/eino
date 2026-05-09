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

	"nhooyr.io/websocket"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsWebSocketRequest(t *testing.T) {
	tests := []struct {
		name     string
		upgrade  string
		expected bool
	}{
		{"websocket upgrade", "websocket", true},
		{"websocket uppercase", "WebSocket", true},
		{"no upgrade", "", false},
		{"other upgrade", "h2c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/chat", nil)
			if tt.upgrade != "" {
				r.Header.Set("Upgrade", tt.upgrade)
			}
			assert.Equal(t, tt.expected, isWebSocketRequest(r))
		})
	}
}

func TestWSWriterE2E(t *testing.T) {
	// Create a test server that handles WebSocket connections
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := newWSWriter(w, r)
		if err != nil {
			t.Logf("newWSWriter error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer ws.Close()

		err = ws.WriteEvent(r.Context(), &Event{
			Type:    EventTypeMessage,
			Content: "hello via ws",
		})
		assert.NoError(t, err)
	}))
	defer s.Close()

	// Connect via WebSocket
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, msg, err := conn.Read(context.Background())
	require.NoError(t, err)
	assert.Contains(t, string(msg), `"type":"message"`)
	assert.Contains(t, string(msg), "hello via ws")
}

func TestSSEWriterWithHttptest(t *testing.T) {
	// httptest.ResponseRecorder implements http.Flusher (since Go 1.20)
	w := httptest.NewRecorder()
	var rw http.ResponseWriter = w
	_, ok := rw.(http.Flusher)
	assert.True(t, ok, "httptest.ResponseRecorder should implement Flusher")
}

func TestExtractCheckpointID(t *testing.T) {
	tests := []struct {
		path     string
		basePath string
		expected string
	}{
		{"/resume/abc-123", "", "abc-123"},
		{"/v1/resume/xyz-456", "/v1", "xyz-456"},
		{"/chat", "", ""},
		{"/resume/", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractCheckpointID(tt.path, tt.basePath))
		})
	}
}

func TestServerOptions(t *testing.T) {
	c := defaultConfig()
	assert.Equal(t, ":8080", c.addr)
	assert.Equal(t, "", c.basePath)

	WithAddr(":9090")(c)
	assert.Equal(t, ":9090", c.addr)

	WithBasePath("/v2/")(c)
	assert.Equal(t, "/v2", c.basePath) // trailing slash trimmed

	WithAllowedOrigins("*")(c)
	assert.Len(t, c.allowedOrigins, 1)

	WithOnEvent(func(ctx context.Context, e *Event) (*Event, error) { return e, nil })(c)
	assert.NotNil(t, c.onEvent)
}
