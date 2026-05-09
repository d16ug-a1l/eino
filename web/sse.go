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
	"net/http"
)

type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSEWriter(w http.ResponseWriter) (*sseWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("web: streaming not supported by this http.ResponseWriter")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return &sseWriter{w: w, flusher: flusher}, nil
}

func (w *sseWriter) WriteEvent(ctx context.Context, event *Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("web: failed to marshal SSE event: %w", err)
	}
	_, err = fmt.Fprintf(w.w, "data: %s\n\n", data)
	if err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

func (w *sseWriter) Close() error {
	return nil
}
