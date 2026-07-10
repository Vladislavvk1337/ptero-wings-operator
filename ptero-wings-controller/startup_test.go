package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestStartupMapperResolve(t *testing.T) {
	mapper := StartupMapper{}
	command, args, err := mapper.Resolve(`java -Xmx{{SERVER_MEMORY}}M -jar "{{SERVER_JARFILE}}"`, map[string]string{
		"SERVER_MEMORY":  "1024",
		"SERVER_JARFILE": "server.jar",
	}, []corev1.EnvVar{{Name: "IGNORED", Value: "noop"}})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(command) != 1 || command[0] != "java" {
		t.Fatalf("command = %v, want [java]", command)
	}
	if len(args) != 3 || args[0] != "-Xmx1024M" || args[1] != "-jar" || args[2] != "server.jar" {
		t.Fatalf("args = %v", args)
	}
}
