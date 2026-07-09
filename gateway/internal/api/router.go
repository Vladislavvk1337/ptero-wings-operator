/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package api wires up the HTTP router for the ptero-wings-gateway.
// All routes are registered under the /api prefix for Wings API compatibility.
package api

import (
	"net/http"

	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/api/handlers"
	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/auth"
	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/console"
	gsk8s "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/k8s"
	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/logs"
)

// Config holds all dependencies needed to build the API router.
type Config struct {
	K8sClient *gsk8s.Client
	// Namespace is the Kubernetes namespace where GameServer CRs are managed.
	Namespace string
	// AuthToken is the static bearer token (or JWT secret) expected from external clients.
	AuthToken string
	// UseJWT switches to JWT HS256 token validation.
	UseJWT bool
}

// NewRouter returns an http.Handler with all Wings-compatible routes registered.
//
// Routes:
//
//	POST   /api/servers                    → create GameServer
//	GET    /api/servers/{uuid}             → get GameServer status
//	DELETE /api/servers/{uuid}             → delete GameServer
//	POST   /api/servers/{uuid}/power       → start / stop / restart
//	GET    /api/servers/{uuid}/resources   → resource stats
//	GET    /api/servers/{uuid}/logs        → recent log lines
//	GET    /api/servers/{uuid}/ws          → WebSocket console
func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()

	logReader := logs.NewReader(cfg.K8sClient.Clientset())
	executor := console.NewExecutor(cfg.K8sClient.Clientset(), cfg.K8sClient.RESTConfig())

	serversH := &handlers.ServersHandler{K8s: cfg.K8sClient, Namespace: cfg.Namespace}
	powerH := &handlers.PowerHandler{K8s: cfg.K8sClient, Namespace: cfg.Namespace}
	resourcesH := &handlers.ResourcesHandler{K8s: cfg.K8sClient, Namespace: cfg.Namespace}
	logsH := &handlers.LogsHandler{K8s: cfg.K8sClient, Reader: logReader, Namespace: cfg.Namespace}
	consoleH := &handlers.ConsoleHandler{
		K8s:       cfg.K8sClient,
		Reader:    logReader,
		Executor:  executor,
		Namespace: cfg.Namespace,
		AuthToken: cfg.AuthToken,
	}

	// API routes wrapped with bearer-token authentication.
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("POST /api/servers", serversH.Create)
	apiMux.HandleFunc("GET /api/servers/{uuid}", serversH.Get)
	apiMux.HandleFunc("DELETE /api/servers/{uuid}", serversH.Delete)
	apiMux.HandleFunc("POST /api/servers/{uuid}/power", powerH.Handle)
	apiMux.HandleFunc("GET /api/servers/{uuid}/resources", resourcesH.Handle)
	apiMux.HandleFunc("GET /api/servers/{uuid}/logs", logsH.Handle)
	apiMux.HandleFunc("GET /api/servers/{uuid}/ws", consoleH.ServeHTTP)

	authCfg := auth.Config{Token: cfg.AuthToken, UseJWT: cfg.UseJWT}
	authedAPI := auth.Middleware(authCfg, apiMux)

	// Top-level mux: health probe is unauthenticated; everything else requires auth.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/api/", authedAPI)

	return mux
}
