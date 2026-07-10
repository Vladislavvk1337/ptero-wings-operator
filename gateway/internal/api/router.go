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

package api

import (
	"net/http"

	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/api/handlers"
	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/auth"
	wingscontroller "github.com/Vladislavvk1337/ptero-wings-operator/ptero-wings-controller"
)

type Config struct {
	Service        wingscontroller.Service
	AuthToken      string
	UseJWT         bool
	AllowedOrigins map[string]struct{}
}

func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()

	serversH := &handlers.ServersHandler{Service: cfg.Service}
	powerH := &handlers.PowerHandler{Service: cfg.Service}
	resourcesH := &handlers.ResourcesHandler{Service: cfg.Service}
	logsH := &handlers.LogsHandler{Service: cfg.Service}
	consoleH := &handlers.ConsoleHandler{
		Service:        cfg.Service,
		AuthToken:      cfg.AuthToken,
		AllowedOrigins: cfg.AllowedOrigins,
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("POST /api/servers", serversH.Create)
	apiMux.HandleFunc("GET /api/servers/{uuid}", serversH.Get)
	apiMux.HandleFunc("DELETE /api/servers/{uuid}", serversH.Delete)
	apiMux.HandleFunc("POST /api/servers/{uuid}/power", powerH.Handle)
	apiMux.HandleFunc("GET /api/servers/{uuid}/resources", resourcesH.Handle)
	apiMux.HandleFunc("GET /api/servers/{uuid}/logs", logsH.Handle)
	apiMux.HandleFunc("GET /api/servers/{uuid}/ws", consoleH.ServeHTTP)

	authedAPI := auth.Middleware(auth.Config{Token: cfg.AuthToken, UseJWT: cfg.UseJWT}, apiMux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/api/", authedAPI)
	return mux
}
