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

package handlers

import (
	"net/http"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	gsk8s "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/k8s"
	gslog "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/logs"
)

// LogsHandler handles GET /api/servers/{uuid}/logs.
type LogsHandler struct {
	K8s       *gsk8s.Client
	Reader    *gslog.Reader
	Namespace string
}

// Handle returns the last N lines of the game server's stdout/stderr log.
// The ?lines=N query parameter controls the number of lines (default 100).
//
//	Response 200: plain-text log lines (UTF-8)
func (h *LogsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}

	// Parse optional ?lines query parameter.
	var lines int64 = 100
	if raw := r.URL.Query().Get("lines"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			lines = n
		}
	}

	ctx := r.Context()

	// Resolve pod name via the GameServer status.
	pod, err := h.K8s.FindPodForGameServer(ctx, h.Namespace, uuid)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data, err := h.Reader.Tail(ctx, h.Namespace, pod.Name, lines)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetching logs: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
