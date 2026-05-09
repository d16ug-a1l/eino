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
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed ui/dist/*
var uiAssets embed.FS

func (s *Server) uiHandler() http.Handler {
	if s.cfg.uiDir != "" {
		return http.FileServer(http.Dir(s.cfg.uiDir))
	}

	dist, err := fs.Sub(uiAssets, "ui/dist")
	if err != nil {
		return http.NotFoundHandler()
	}

	// Check if the dist directory contains a real build (index.html beyond .gitkeep)
	hasUI := false
	fs.WalkDir(dist, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if filepath.Base(path) == "index.html" {
			hasUI = true
		}
		return nil
	})

	if !hasUI {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "UI not built — run 'cd web/ui && npm install && npm run build'", http.StatusNotFound)
		})
	}

	fileServer := http.FileServer(http.FS(dist))
	return fileServer
}

// WithEnableUI controls whether to serve the chat UI at the root path.
// Default is true. Set to false to only expose the API endpoints.
func WithEnableUI(enabled bool) ServerOption {
	return func(c *serverConfig) {
		c.enableUI = enabled
	}
}

// WithUIDir sets a local directory to serve UI files from, useful during development.
// When set, the embedded UI is bypassed and files are served directly from this directory.
func WithUIDir(dir string) ServerOption {
	return func(c *serverConfig) {
		c.uiDir = dir
	}
}

// WriteStaticUI writes the embedded UI files to a local directory.
// This is useful for serving the UI via an external web server (e.g., nginx)
// or for inspecting the built-in UI.
func WriteStaticUI(dir string) error {
	return os.MkdirAll(dir, 0755)
}
