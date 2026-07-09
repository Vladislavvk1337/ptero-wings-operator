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
	"fmt"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	gsk8s "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/k8s"
)

// powerAction is the Wings-compatible power action value.
type powerAction string

const (
	powerStart   powerAction = "start"
	powerStop    powerAction = "stop"
	powerRestart powerAction = "restart"
	powerKill    powerAction = "kill"
)

// powerRequest is the JSON body for POST /api/servers/{uuid}/power.
type powerRequest struct {
	Action powerAction `json:"action"`
}

// PowerHandler handles POST /api/servers/{uuid}/power.
type PowerHandler struct {
	K8s       *gsk8s.Client
	Namespace string
}

// Handle processes a power action request.
//
//	Request body: {"action": "start"|"stop"|"restart"|"kill"}
//	Response 204: no body
func (h *PowerHandler) Handle(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}

	var req powerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	ctx := r.Context()
	var err error

	switch req.Action {
	case powerStart:
		// Resume: set suspended=false
		err = h.K8s.SetSuspended(ctx, h.Namespace, uuid, false)

	case powerStop, powerKill:
		// Suspend: set suspended=true (scales StatefulSet to 0)
		err = h.K8s.SetSuspended(ctx, h.Namespace, uuid, true)

	case powerRestart:
		// Suspend then immediately resume — the operator will reconcile
		// both patches. In practice the StatefulSet scales to 0 then back to 1.
		if err = h.K8s.SetSuspended(ctx, h.Namespace, uuid, true); err == nil {
			err = h.K8s.SetSuspended(ctx, h.Namespace, uuid, false)
		}

	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown action: %q", req.Action))
		return
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
