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
	"strings"
)

type serverConfig struct {
	addr           string
	basePath       string
	allowedOrigins []string
	onEvent        func(context.Context, *Event) (*Event, error)
	enableUI       bool
	uiDir          string
}

func defaultConfig() *serverConfig {
	return &serverConfig{
		addr:     ":8080",
		enableUI: true,
	}
}

// ServerOption configures a Server.
type ServerOption func(*serverConfig)

// WithAddr sets the listen address. Default ":8080".
func WithAddr(addr string) ServerOption {
	return func(c *serverConfig) {
		c.addr = addr
	}
}

// WithBasePath sets a URL path prefix for all routes. Default "".
// Example: WithBasePath("/v1") creates routes /v1/chat, /v1/resume/{id}.
func WithBasePath(path string) ServerOption {
	return func(c *serverConfig) {
		c.basePath = strings.TrimRight(path, "/")
	}
}

// WithAllowedOrigins sets CORS allowed origins for browser clients.
// Pass "*" to allow all origins.
func WithAllowedOrigins(origins ...string) ServerOption {
	return func(c *serverConfig) {
		c.allowedOrigins = origins
	}
}

// WithOnEvent registers a callback invoked before each event is sent.
// The callback can inspect, log, or modify the event. Returning nil
// from the callback skips sending that event.
func WithOnEvent(fn func(context.Context, *Event) (*Event, error)) ServerOption {
	return func(c *serverConfig) {
		c.onEvent = fn
	}
}
