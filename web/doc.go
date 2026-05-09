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

// Package web provides an embedded HTTP server for exposing ADK Agents via
// SSE (Server-Sent Events) and WebSocket.
//
// The simplest usage:
//
//	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true})
//	web.NewServer(runner).Start(ctx)
//
// The server exposes:
//   - GET/POST /chat        — send messages, receive streaming agent events
//   - POST      /resume/{id} — resume an interrupted execution
//   - GET       /health      — health check
//
// The /chat and /resume endpoints automatically detect WebSocket clients via
// the Upgrade header and upgrade the connection when present. All other
// clients receive SSE.
package web
