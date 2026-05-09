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
)

func (s *Server) resumeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	checkpointID := extractCheckpointID(r.URL.Path, s.cfg.basePath)
	if checkpointID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "checkpoint ID is required"})
		return
	}

	var params adk.ResumeParams
	if r.Body != nil {
		var req resumeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			params.Targets = req.Targets
		}
	}

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
		log.Printf("web: failed to create transport for resume: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create transport"})
		return
	}
	defer writer.Close()

	iter, err := s.runner.ResumeWithParams(r.Context(), checkpointID, &params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resumeID := uuid.NewString()
	if err := processEvents(r.Context(), writer, iter, resumeID); err != nil {
		if !isClientDisconnect(err) {
			log.Printf("web: error processing resume events: %v", err)
		}
	}
}

type resumeRequest struct {
	Targets map[string]any `json:"targets"`
}

// extractCheckpointID extracts the checkpoint ID from a URL path like /resume/{id}
// or /{basePath}/resume/{id}.
func extractCheckpointID(path string, basePath string) string {
	prefix := basePath + "/resume/"
	if !strings.HasPrefix(path, prefix) {
		prefix = "/resume/"
		if !strings.HasPrefix(path, prefix) {
			return ""
		}
	}
	return strings.TrimPrefix(path, prefix)
}
