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

import "github.com/cloudwego/eino/schema"

// Event is the wire-format JSON structure sent to clients over SSE and WebSocket.
type Event struct {
	Type         string            `json:"type"`
	AgentName    string            `json:"agent_name,omitempty"`
	RunPath      string            `json:"run_path,omitempty"`
	Content      string            `json:"content,omitempty"`
	ToolCalls    []schema.ToolCall `json:"tool_calls,omitempty"`
	ActionType   string            `json:"action_type,omitempty"`
	Error        string            `json:"error,omitempty"`
	CheckPointID string            `json:"checkpoint_id,omitempty"`
}

// Event type constants.
const (
	EventTypeMessage         = "message"
	EventTypeToolResult      = "tool_result"
	EventTypeStreamChunk     = "stream_chunk"
	EventTypeToolResultChunk = "tool_result_chunk"
	EventTypeToolCalls       = "tool_calls"
	EventTypeAction          = "action"
	EventTypeError           = "error"
)

// Action type constants for the ActionType field in action events.
const (
	ActionTypeTransfer   = "transfer"
	ActionTypeInterrupted = "interrupted"
	ActionTypeExit       = "exit"
	ActionTypeBreakLoop  = "break_loop"
)
