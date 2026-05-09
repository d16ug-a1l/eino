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
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/adk"
)

// Server is an embedded HTTP server that exposes an ADK Runner via SSE and WebSocket.
type Server struct {
	runner     *adk.Runner
	cfg        *serverConfig
	httpServer *http.Server
}

// NewServer creates a Server that wraps an adk.Runner.
// The Runner must have EnableStreaming set to true.
func NewServer(runner *adk.Runner, opts ...ServerOption) *Server {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}
	return &Server{runner: runner, cfg: cfg}
}

// Start begins serving HTTP requests. It blocks until ctx is cancelled,
// then performs a graceful shutdown.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	p := s.cfg.basePath

	mux.HandleFunc(p+"/chat", s.wrapCORS(s.chatHandler))
	mux.HandleFunc(p+"/resume/", s.wrapCORS(s.resumeHandler))
	mux.HandleFunc(p+"/health", s.wrapCORS(s.healthHandler))

	if s.cfg.enableUI {
		uiHandler := s.uiHandler()
		mux.Handle(p+"/", uiHandler)
	}

	s.httpServer = &http.Server{
		Addr:    s.cfg.addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("web: shutdown error: %v", err)
		}
	}()

	log.Printf("web: serving on %s%s", s.cfg.addr, p)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) wrapCORS(next http.HandlerFunc) http.HandlerFunc {
	if len(s.cfg.allowedOrigins) == 0 {
		return next
	}
	allowAll := false
	for _, origin := range s.cfg.allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowed := range s.cfg.allowedOrigins {
					if allowed == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Upgrade")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
