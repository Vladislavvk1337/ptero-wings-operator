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

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PanelClient interface {
	PushResources(ctx context.Context, payload *PanelResourcePayload) error
}

type HTTPPanelClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewPanelClient(baseURL, token string) PanelClient {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(token) == "" {
		return noopPanelClient{}
	}
	return &HTTPPanelClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPPanelClient) PushResources(ctx context.Context, payload *PanelResourcePayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal panel payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/client/servers/"+payload.ServerID+"/resources", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build panel request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send panel request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("panel returned status %s", resp.Status)
	}
	return nil
}

type noopPanelClient struct{}

func (noopPanelClient) PushResources(context.Context, *PanelResourcePayload) error { return nil }
