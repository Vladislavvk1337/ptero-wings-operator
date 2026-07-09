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

// Package console provides helpers for executing commands inside a game server pod
// via the Kubernetes exec API.
package console

import (
	"context"
	"fmt"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

const containerName = "gameserver"

// Executor wraps a Kubernetes clientset and REST config to provide pod exec support.
type Executor struct {
	clientset kubernetes.Interface
	restCfg   *rest.Config
}

// NewExecutor creates an Executor backed by the provided clientset and REST config.
func NewExecutor(cs kubernetes.Interface, cfg *rest.Config) *Executor {
	return &Executor{clientset: cs, restCfg: cfg}
}

// ExecStream runs cmd inside the named pod, wiring stdin/stdout/stderr to the provided streams.
// Use context cancellation to terminate the exec session.
func (e *Executor) ExecStream(ctx context.Context, namespace, podName string, cmd []string, stdin io.Reader, stdout, stderr io.Writer) error {
	req := e.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   cmd,
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(e.restCfg, http.MethodPost, req.URL())
	if err != nil {
		return fmt.Errorf("creating SPDY executor: %w", err)
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
}
