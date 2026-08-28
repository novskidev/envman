package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComposeEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docker-compose.yml")
	content := `services:
  web:
    image: nginx
    environment:
      PORT: "8080"
      DEBUG: "true"
      HOST_TOKEN:   # host passthrough
  worker:
    image: alpine
    environment:
      - REDIS_URL=redis://localhost:6379
      - QUEUE_NAME
  db:
    image: postgres
    # no environment section
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	svcs, err := ComposeEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(svcs) != 2 {
		t.Fatalf("want 2 services with environment, got %d", len(svcs))
	}
	// sorted: web, worker (db has no environment section, filtered out)
	if svcs[0].Name != "web" {
		t.Fatalf("web sorted first, got %s", svcs[0].Name)
	}
	if len(svcs[0].Environment) != 3 {
		t.Fatalf("web: want 3 env entries, got %d: %v", len(svcs[0].Environment), svcs[0].Environment)
	}
	webKeys := make(map[string]bool, len(svcs[0].Environment))
	for _, e := range svcs[0].Environment {
		webKeys[keyOfEnvEntry(e)] = true
	}
	workerKeys := make(map[string]bool, len(svcs[1].Environment))
	for _, e := range svcs[1].Environment {
		workerKeys[keyOfEnvEntry(e)] = true
	}
	for key, want := range map[string]bool{"DEBUG": true, "PORT": true, "HOST_TOKEN": true} {
		if webKeys[key] != want {
			t.Errorf("web key %q missing", key)
		}
	}
	for key, want := range map[string]bool{"REDIS_URL": true, "QUEUE_NAME": true} {
		if workerKeys[key] != want {
			t.Errorf("worker key %q missing", key)
		}
	}
}