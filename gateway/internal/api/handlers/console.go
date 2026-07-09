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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	gsconsole "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/console"
	gsk8s "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/k8s"
	gslog "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/logs"
)

// wsUpgrader upgrades HTTP connections to WebSocket.
// CheckOrigin is intentionally permissive here; callers should set
// AllowedOrigins based on their deployment if needed.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

// wsMessage is the JSON envelope used on the WebSocket console channel.
// It mirrors the Wings WebSocket protocol closely enough for panel compatibility.
type wsMessage struct {
	// Event is one of: "auth", "send command", "console output", "status".
	Event string `json:"event"`
	// Args contains event-specific payload strings.
	Args []string `json:"args,omitempty"`
}

// ConsoleHandler handles WS /api/servers/{uuid}/ws — the live console WebSocket.
//
// Protocol (client → server):
//
//	{"event":"auth","args":["<bearer-token>"]}
//	{"event":"send command","args":["say Hello"]}
//
// Protocol (server → client):
//
//	{"event":"console output","args":["<log line>"]}
//	{"event":"status","args":["<phase>"]}
type ConsoleHandler struct {
	K8s       *gsk8s.Client
	Reader    *gslog.Reader
	Executor  *gsconsole.Executor
	Namespace string
	// AuthToken is validated against the token sent in the "auth" WebSocket event.
	AuthToken string
}

// ServeHTTP upgrades to WebSocket and starts bi-directional log + exec streaming.
func (h *ConsoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Step 1: wait for auth event.
	if !h.waitForAuth(conn) {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth failed"))
		return
	}

	// Step 2: find the game server pod.
	pod, err := h.K8s.FindPodForGameServer(ctx, h.Namespace, uuid)
	if err != nil {
		h.sendEvent(conn, "status", "error: "+err.Error())
		return
	}

	// Step 3: stream logs to the WebSocket in a goroutine.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pr, pw := io.Pipe()
		defer pw.Close()

		// Feed log lines to the WebSocket.
		go func() {
			buf := make([]byte, 4096)
			for {
				n, readErr := pr.Read(buf)
				if n > 0 {
					h.sendEvent(conn, "console output", string(buf[:n]))
				}
				if readErr != nil {
					return
				}
			}
		}()

		if streamErr := h.Reader.Stream(ctx, h.Namespace, pod.Name, pw); streamErr != nil {
			if ctx.Err() == nil {
				slog.Warn("log stream ended", "err", streamErr)
			}
		}
	}()

	// Step 4: read commands from WebSocket and exec them in the pod.
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			cancel()
			break
		}

		var msg wsMessage
		if jsonErr := json.Unmarshal(raw, &msg); jsonErr != nil {
			continue
		}

		if msg.Event == "send command" && len(msg.Args) > 0 {
			cmd := msg.Args[0]
			go func(c string) {
				stdin := bytes.NewBufferString(c + "\n")
				var out bytes.Buffer
				execErr := h.Executor.ExecStream(ctx, h.Namespace, pod.Name,
					[]string{"/bin/sh", "-c", c}, stdin, &out, &out)
				if execErr != nil && ctx.Err() == nil {
					slog.Warn("exec error", "cmd", c, "err", execErr)
				}
				if out.Len() > 0 {
					h.sendEvent(conn, "console output", out.String())
				}
			}(cmd)
		}
	}

	wg.Wait()
}

// waitForAuth reads the first WebSocket message and validates the auth event token.
func (h *ConsoleHandler) waitForAuth(conn *websocket.Conn) bool {
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return false
	}
	var msg wsMessage
	if err := json.Unmarshal(raw, &msg); err != nil || msg.Event != "auth" || len(msg.Args) == 0 {
		return false
	}
	return msg.Args[0] == h.AuthToken
}

// sendEvent writes a wsMessage JSON frame to the WebSocket connection.
func (h *ConsoleHandler) sendEvent(conn *websocket.Conn, event string, args ...string) {
	msg := wsMessage{Event: event, Args: args}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if writeErr := conn.WriteMessage(websocket.TextMessage, data); writeErr != nil {
		slog.Debug("ws write error", "err", writeErr, "event", fmt.Sprintf("%q", event))
	}
}
