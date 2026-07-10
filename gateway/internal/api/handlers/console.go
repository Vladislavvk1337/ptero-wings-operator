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

	wingscontroller "github.com/Vladislavvk1337/ptero-wings-operator/ptero-wings-controller"
)

var wsUpgrader = websocket.Upgrader{}

func newUpgrader(allowedOrigins map[string]struct{}) websocket.Upgrader {
	if len(allowedOrigins) == 0 {
		return wsUpgrader
	}
	return websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		_, ok := allowedOrigins[r.Header.Get("Origin")]
		return ok
	}}
}

type wsMessage struct {
	Event string   `json:"event"`
	Args  []string `json:"args,omitempty"`
}

type ConsoleHandler struct {
	Service        wingscontroller.Service
	AuthToken      string
	AllowedOrigins map[string]struct{}
}

func (h *ConsoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uuid := serverIDFromPath(r)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "missing server uuid")
		return
	}
	conn, err := newUpgrader(h.AllowedOrigins).Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	if !h.waitForAuth(conn) {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth failed"))
		return
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pr, pw := io.Pipe()
		defer pw.Close()
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
		if streamErr := h.Service.StreamLogs(ctx, uuid, pw); streamErr != nil && ctx.Err() == nil {
			slog.Warn("log stream ended", "err", streamErr)
		}
	}()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			cancel()
			break
		}
		var msg wsMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if msg.Event == "send command" && len(msg.Args) > 0 {
			command := msg.Args[0]
			go func() {
				var out bytes.Buffer
				execErr := h.Service.ExecCommand(ctx, uuid, command, &out, &out)
				if execErr != nil && ctx.Err() == nil {
					slog.Warn("exec error", "cmd", command, "err", execErr)
				}
				if out.Len() > 0 {
					h.sendEvent(conn, "console output", out.String())
				}
			}()
		}
	}
	wg.Wait()
}

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

func (h *ConsoleHandler) sendEvent(conn *websocket.Conn, event string, args ...string) {
	data, err := json.Marshal(wsMessage{Event: event, Args: args})
	if err != nil {
		return
	}
	if writeErr := conn.WriteMessage(websocket.TextMessage, data); writeErr != nil {
		slog.Debug("ws write error", "err", writeErr, "event", fmt.Sprintf("%q", event))
	}
}
