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
	"fmt"
	"strings"
	"unicode"

	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/Vladislavvk1337/ptero-wings-operator/api/v1alpha1"
)

type StartupMapper struct{}

func (StartupMapper) Resolve(raw string, variables map[string]string, env []corev1.EnvVar) ([]string, []string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil, nil
	}
	replaced := expandStartup(raw, mergedVariables(variables, env))
	tokens, err := splitCommand(replaced)
	if err != nil {
		return nil, nil, err
	}
	if len(tokens) == 0 {
		return nil, nil, nil
	}
	return []string{tokens[0]}, tokens[1:], nil
}

func expandStartup(raw string, vars map[string]string) string {
	replaced := raw
	for key, value := range vars {
		replaced = strings.ReplaceAll(replaced, "{{"+key+"}}", value)
	}
	return replaced
}

func mergedVariables(variables map[string]string, env []corev1.EnvVar) map[string]string {
	merged := make(map[string]string, len(variables)+len(env))
	for _, item := range env {
		if item.Value != "" {
			merged[item.Name] = item.Value
		}
	}
	for key, value := range variables {
		merged[key] = value
	}
	return merged
}

func splitCommand(raw string) ([]string, error) {
	var (
		tokens  []string
		current strings.Builder
		quote   rune
		escaped bool
	)

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for _, ch := range raw {
		switch {
		case escaped:
			current.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case quote != 0:
			if ch == quote {
				quote = 0
				continue
			}
			current.WriteRune(ch)
		case ch == '\'' || ch == '"':
			quote = ch
		case unicode.IsSpace(ch):
			flush()
		default:
			current.WriteRune(ch)
		}
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape in startup command")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in startup command")
	}
	flush()
	return tokens, nil
}

func ApplyStartup(gs *v1alpha1.GameServer, mapper StartupMapper) error {
	if gs.Spec.Runtime.Startup == nil {
		return nil
	}
	command, args, err := mapper.Resolve(gs.Spec.Runtime.Startup.Raw, gs.Spec.Runtime.Startup.Variables, gs.Spec.Game.Env)
	if err != nil {
		return err
	}
	if len(command) > 0 {
		gs.Spec.Game.Command = command
	}
	if args != nil {
		gs.Spec.Game.Args = args
	}
	return nil
}
