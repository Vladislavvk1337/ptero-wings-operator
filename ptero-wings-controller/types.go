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
	"context"
	"io"

	"k8s.io/client-go/rest"
)

type Config struct {
	RESTConfig  *rest.Config
	Namespace   string
	PanelURL    string
	PanelToken  string
	NodePortMin int32
	NodePortMax int32
}

type AllocationRequest struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	BindPort int    `json:"bind_port"`
}

type ResourceLimits struct {
	CPU    float64 `json:"cpu"`
	Memory int64   `json:"memory"`
}

type CreateServerRequest struct {
	UUID             string              `json:"uuid"`
	GameType         string              `json:"game_type"`
	Image            string              `json:"image"`
	Environment      map[string]string   `json:"environment"`
	Limits           ResourceLimits      `json:"limits"`
	DiskMB           int64               `json:"disk"`
	Allocations      []AllocationRequest `json:"allocations"`
	Startup          string              `json:"startup,omitempty"`
	StartupVariables map[string]string   `json:"startup_variables,omitempty"`
	PortRangeStart   int32               `json:"port_range_start,omitempty"`
	PortRangeCount   int32               `json:"port_range_count,omitempty"`
	PortRangeBase    int32               `json:"port_range_base,omitempty"`
	ExternalServerID string              `json:"external_server_id,omitempty"`
}

type ServerView struct {
	ID       string `json:"id"`
	Phase    string `json:"phase"`
	Endpoint string `json:"endpoint,omitempty"`
}

type ResourcesResponse struct {
	State          string            `json:"state"`
	CPUAbsolute    float64           `json:"cpu_absolute"`
	MemoryBytes    int64             `json:"memory_bytes"`
	DiskBytes      int64             `json:"disk_bytes"`
	NetworkRxBytes int64             `json:"network_rx_bytes"`
	NetworkTxBytes int64             `json:"network_tx_bytes"`
	UptimeSeconds  int64             `json:"uptime_seconds"`
	Cluster        *ClusterResources `json:"cluster,omitempty"`
}

type ClusterResources struct {
	NodeCount   int   `json:"node_count"`
	PodCount    int   `json:"pod_count"`
	CPUMillis   int64 `json:"cpu_millis_capacity"`
	MemoryBytes int64 `json:"memory_bytes_capacity"`
}

type PowerAction string

const (
	PowerActionStart   PowerAction = "start"
	PowerActionStop    PowerAction = "stop"
	PowerActionRestart PowerAction = "restart"
	PowerActionKill    PowerAction = "kill"
)

type Service interface {
	CreateServer(ctx context.Context, req *CreateServerRequest) (*ServerView, error)
	GetServer(ctx context.Context, id string) (*ServerView, error)
	DeleteServer(ctx context.Context, id string) error
	SetPowerState(ctx context.Context, id string, action PowerAction) error
	GetResources(ctx context.Context, id string) (*ResourcesResponse, error)
	TailLogs(ctx context.Context, id string, lines int64) ([]byte, error)
	StreamLogs(ctx context.Context, id string, w io.Writer) error
	ExecCommand(ctx context.Context, id, command string, stdout, stderr io.Writer) error
}

type PanelResourcePayload struct {
	ServerID string             `json:"server_id"`
	Stats    *ResourcesResponse `json:"stats"`
}
