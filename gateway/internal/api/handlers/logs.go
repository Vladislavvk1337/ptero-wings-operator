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

	wingscontroller "github.com/Vladislavvk1337/ptero-wings-operator/ptero-wings-controller"
)

type LogsHandler struct {
	Service wingscontroller.Service
}

func (h *LogsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}
	var lines int64 = 100
	if raw := r.URL.Query().Get("lines"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			lines = parsed
		}
	}
	data, err := h.Service.TailLogs(r.Context(), uuid, lines)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetching logs: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
