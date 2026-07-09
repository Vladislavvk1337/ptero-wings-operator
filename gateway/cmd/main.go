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

// Package main is the entry point for the ptero-wings-gateway.
//
// The gateway acts as a Wings-compatible HTTP/WebSocket API server.
// It maps Wings API calls from external clients (e.g. a Pterodactyl panel)
// to Kubernetes GameServer CRs managed by the ptero-wings-operator.
//
// Configuration is read from environment variables:
//
//	GATEWAY_ADDR       - listen address, default ":8090"
//	GATEWAY_TOKEN      - bearer token expected from clients (required)
//	GATEWAY_USE_JWT    - set to "true" to validate tokens as HS256 JWTs
//	GAMESERVER_NS      - Kubernetes namespace for GameServer CRs, default "game-servers"
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/api"
	gsk8s "github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/k8s"
)

func main() {
	var addr string
	var kubeconfig string
	var tlsCert string
	var tlsKey string

	flag.StringVar(&addr, "addr", envOrDefault("GATEWAY_ADDR", ":8090"), "Listen address.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (empty = in-cluster).")
	flag.StringVar(&tlsCert, "tls-cert", "", "Path to TLS certificate file (enables HTTPS).")
	flag.StringVar(&tlsKey, "tls-key", "", "Path to TLS private key file (enables HTTPS).")
	flag.Parse()

	token := os.Getenv("GATEWAY_TOKEN")
	if token == "" {
		slog.Error("GATEWAY_TOKEN is required")
		os.Exit(1)
	}

	ns := envOrDefault("GAMESERVER_NS", "game-servers")
	useJWT := os.Getenv("GATEWAY_USE_JWT") == "true"

	// Build Kubernetes REST config.
	restCfg, err := buildRESTConfig(kubeconfig)
	if err != nil {
		slog.Error("building kubernetes config", "err", err)
		os.Exit(1)
	}

	k8sClient, err := gsk8s.NewClient(restCfg)
	if err != nil {
		slog.Error("building kubernetes client", "err", err)
		os.Exit(1)
	}

	handler := api.NewRouter(api.Config{
		K8sClient: k8sClient,
		Namespace: ns,
		AuthToken: token,
		UseJWT:    useJWT,
	})

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // 0 = no write deadline; required for WebSocket / streaming log responses.
		IdleTimeout:  120 * time.Second,
		// Disable HTTP/2 by default to avoid CVE-2023-44487 / CVE-2023-39325.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	// Graceful shutdown on SIGTERM / SIGINT.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if tlsCert != "" && tlsKey != "" {
			slog.Info("ptero-wings-gateway starting (HTTPS)", "addr", addr)
			if listenErr := srv.ListenAndServeTLS(tlsCert, tlsKey); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
				slog.Error("server error", "err", listenErr)
				os.Exit(1)
			}
		} else {
			slog.Info("ptero-wings-gateway starting (HTTP)", "addr", addr)
			if listenErr := srv.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
				slog.Error("server error", "err", listenErr)
				os.Exit(1)
			}
		}
	}()

	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if shutErr := srv.Shutdown(ctx); shutErr != nil {
		slog.Error("shutdown error", "err", shutErr)
	}
}

func buildRESTConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg, nil
	}
	// Fall back to default kubeconfig location.
	loadRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadRules, &clientcmd.ConfigOverrides{}).ClientConfig()
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
