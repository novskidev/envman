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
	web := svcs[0]
	if web.Name != "db" { // sorted: db, web, worker
		t.Fatalf("db sorted first, got %s", web.Name)
	}
	if len(svcs[1].Environment) != 3 {
		t.Fatalf("web: want 3 env entries, got %d: %v", len(svcs[1].Environment), svcs[1].Environment)
	}
	if keyOfEnvEntry(svcs[1].Environment[0]) != "DEBUG" {
		t.Errorf("web env[0] key = %q, want DEBUG", svcs[1].Environment[0])
	}
	if keyOfEnvEntry(svcs[1].Environment[2]) != "HOST_TOKEN" {
		t.Errorf("web passthrough key = %q, want HOST_TOKEN", svcs[1].Environment[2])
	}
	if keyOfEnvEntry(svcs[2].Environment[1]) != "QUEUE_NAME" {
		t.Errorf("worker key = %q, want QUEUE_NAME", svcs[2].Environment[1])
	}
}