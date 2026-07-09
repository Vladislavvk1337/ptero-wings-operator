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

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	gsk8s "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/k8s"
)

// ResourcesHandler handles GET /api/servers/{uuid}/resources.
type ResourcesHandler struct {
	K8s       *gsk8s.Client
	Namespace string
}

// Handle returns a Wings-compatible resource stats snapshot.
//
//	Response 200: {"cpu_absolute": 0.5, "memory_bytes": 1073741824, "state": "Running"}
func (h *ResourcesHandler) Handle(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}

	stats, err := h.K8s.GetResourceStats(r.Context(), h.Namespace, uuid)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
