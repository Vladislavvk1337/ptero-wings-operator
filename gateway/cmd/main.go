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
	"strconv"
	"strings"
	"syscall"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Vladislavvk1337/ptero-wings-operator/gateway/internal/api"
	wingscontroller "github.com/Vladislavvk1337/ptero-wings-operator/ptero-wings-controller"
)

func main() {
	var addr, kubeconfig, tlsCert, tlsKey string
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
	useJWT := os.Getenv("GATEWAY_USE_JWT") == "true"
	namespace := envOrDefault("GAMESERVER_NS", "game-servers")
	allowedOrigins := parseAllowedOrigins(os.Getenv("GATEWAY_ALLOWED_ORIGINS"))
	restCfg, err := buildRESTConfig(kubeconfig)
	if err != nil {
		slog.Error("building kubernetes config", "err", err)
		os.Exit(1)
	}
	service, err := wingscontroller.NewService(wingscontroller.Config{
		RESTConfig:  restCfg,
		Namespace:   namespace,
		PanelURL:    os.Getenv("PANEL_URL"),
		PanelToken:  os.Getenv("PANEL_TOKEN"),
		NodePortMin: envOrDefaultInt32("NODEPORT_MIN", 30000),
		NodePortMax: envOrDefaultInt32("NODEPORT_MAX", 31000),
	})
	if err != nil {
		slog.Error("building controller service", "err", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr:         addr,
		Handler:      api.NewRouter(api.Config{Service: service, AuthToken: token, UseJWT: useJWT, AllowedOrigins: allowedOrigins}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		if tlsCert != "" && tlsKey != "" {
			slog.Info("ptero-wings-gateway starting (HTTPS)", "addr", addr)
			if listenErr := server.ListenAndServeTLS(tlsCert, tlsKey); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
				slog.Error("server error", "err", listenErr)
				os.Exit(1)
			}
			return
		}
		slog.Info("ptero-wings-gateway starting (HTTP)", "addr", addr)
		if listenErr := server.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			slog.Error("server error", "err", listenErr)
			os.Exit(1)
		}
	}()
	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func buildRESTConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
}

func envOrDefault(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

func envOrDefaultInt32(key string, def int32) int32 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
			return int32(parsed)
		}
	}
	return def
}

func parseAllowedOrigins(s string) map[string]struct{} {
	if s == "" {
		return nil
	}
	set := make(map[string]struct{})
	for _, origin := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	return set
}
