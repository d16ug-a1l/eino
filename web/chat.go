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
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func isWebSocketRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	var query string
	var messages []adk.Message

	switch r.Method {
	case http.MethodGet:
		query = r.URL.Query().Get("query")
		if query == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter is required"})
			return
		}
	case http.MethodPost:
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		query = req.Query
		messages = req.toMessages()
		if query == "" && len(messages) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query or messages is required"})
			return
		}
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	checkpointID := uuid.NewString()

	var (
		writer EventWriter
		err    error
	)

	if isWebSocketRequest(r) {
		writer, err = newWSWriter(w, r)
	} else {
		writer, err = newSSEWriter(w)
	}
	if err != nil {
		log.Printf("web: failed to create transport: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create transport"})
		return
	}
	defer writer.Close()

	var iter *adk.AsyncIterator[*adk.AgentEvent]
	if query != "" {
		iter = s.runner.Query(r.Context(), query, adk.WithCheckPointID(checkpointID))
	} else {
		iter = s.runner.Run(r.Context(), messages, adk.WithCheckPointID(checkpointID))
	}

	if err := processEvents(r.Context(), writer, iter, checkpointID); err != nil {
		if !isClientDisconnect(err) {
			log.Printf("web: error processing events: %v", err)
		}
	}
}

func isClientDisconnect(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection") || strings.Contains(errStr, "closed") ||
		strings.Contains(errStr, "canceled") || strings.Contains(errStr, "broken pipe")
}

// chatRequest is the JSON body for POST /chat.
type chatRequest struct {
	Query    string               `json:"query"`
	Messages []chatRequestMessage `json:"messages"`
}

type chatRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (r *chatRequest) toMessages() []adk.Message {
	var msgs []adk.Message
	for _, m := range r.Messages {
		switch strings.ToLower(m.Role) {
		case "user":
			msgs = append(msgs, schema.UserMessage(m.Content))
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
		case "system":
			msgs = append(msgs, schema.SystemMessage(m.Content))
		case "tool":
			msgs = append(msgs, schema.ToolMessage(m.Content, ""))
		}
	}
	return msgs
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
