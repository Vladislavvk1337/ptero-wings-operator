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
	"encoding/json"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	gsk8s "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/k8s"
)

// ServersHandler handles server lifecycle endpoints.
type ServersHandler struct {
	K8s       *gsk8s.Client
	Namespace string
}

// Create handles POST /api/servers — creates a new GameServer CR.
//
//	Request body: gsk8s.CreateServerRequest (JSON)
//	Response 201: {"id": "<uuid>"}
func (h *ServersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req gsk8s.CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.UUID == "" {
		writeError(w, http.StatusBadRequest, "uuid is required")
		return
	}
	if req.Image == "" {
		writeError(w, http.StatusBadRequest, "image is required")
		return
	}

	gs := gsk8s.DefaultGameServerFrom(&req, h.Namespace)
	if err := h.K8s.CreateGameServer(r.Context(), gs); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeError(w, http.StatusConflict, "server already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": req.UUID})
}

// Delete handles DELETE /api/servers/{uuid} — removes the GameServer CR.
//
//	Response 204: no body
func (h *ServersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}

	if err := h.K8s.DeleteGameServer(r.Context(), h.Namespace, uuid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Get handles GET /api/servers/{uuid} — returns server phase and endpoint.
//
//	Response 200: {"id": "…", "phase": "…", "endpoint": "…"}
func (h *ServersHandler) Get(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}

	gs, err := h.K8s.GetGameServer(r.Context(), h.Namespace, uuid)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":       gs.Name,
		"phase":    string(gs.Status.Phase),
		"endpoint": gs.Status.Endpoint,
	})
}
