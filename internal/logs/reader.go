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

package logs

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultTailLines = 100
	containerName    = "gameserver"
)

type Reader struct {
	clientset kubernetes.Interface
}

func NewReader(cs kubernetes.Interface) *Reader {
	return &Reader{clientset: cs}
}

func (r *Reader) Tail(ctx context.Context, namespace, podName string, n int64) ([]byte, error) {
	if n <= 0 {
		n = defaultTailLines
	}
	req := r.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &n,
	})
	rc, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("opening log stream: %w", err)
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (r *Reader) Stream(ctx context.Context, namespace, podName string, w io.Writer) error {
	req := r.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,
	})
	rc, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("opening follow stream: %w", err)
	}
	defer rc.Close()

	buf := make([]byte, 4096)
	for {
		n, readErr := rc.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
