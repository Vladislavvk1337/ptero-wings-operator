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

	wingscontroller "github.com/Vladislavvk1337/ptero-wings-operator/ptero-wings-controller"
)

type ServersHandler struct {
	Service wingscontroller.Service
}

func (h *ServersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req wingscontroller.CreateServerRequest
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
	server, err := h.Service.CreateServer(r.Context(), &req)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeError(w, http.StatusConflict, "server already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": server.ID})
}

func (h *ServersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}
	if err := h.Service.DeleteServer(r.Context(), uuid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServersHandler) Get(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}
	server, err := h.Service.GetServer(r.Context(), uuid)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, server)
}
