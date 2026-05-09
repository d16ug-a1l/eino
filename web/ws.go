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
	"net/http"

	"nhooyr.io/websocket"
)

type wsWriter struct {
	conn *websocket.Conn
}

func newWSWriter(w http.ResponseWriter, r *http.Request) (*wsWriter, error) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &wsWriter{conn: conn}, nil
}

func (w *wsWriter) WriteEvent(ctx context.Context, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return w.conn.Write(ctx, websocket.MessageText, data)
}

func (w *wsWriter) Close() error {
	return w.conn.Close(websocket.StatusNormalClosure, "closing")
}
