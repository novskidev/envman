package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestScanSecrets(t *testing.T) {
	f := ParseEnv("AWS_KEY=AKIAIOSFODNN7EXAMPLE\n" + // AWS key pattern
		"GITHUB_TOKEN=ghp_1234567890123456789012345678901234567890\n" +
		"JWT=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c\n" +
		"PORT=8080\n" + // safe
		"API_URL=https://example.com\n" + // safe
		"YOUR_API_KEY=" + "sk_" + "live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n" + // fake stripe-shaped value
		"DB_PASSWORD=supersecret\n") // name heuristic

	findings := ScanSecrets(f)
	got := map[string]string{}
	for _, fd := range findings {
		got[fd.Key] = fd.Probe
	}

	wantKeys := []string{"AWS_KEY", "GITHUB_TOKEN", "JWT", "YOUR_API_KEY", "DB_PASSWORD"}
	if len(findings) != len(wantKeys) {
		t.Fatalf("want %d findings, got %d: %+v", len(wantKeys), len(findings), findings)
	}
	for _, k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Errorf("expected finding for %s, got %+v", k, got)
		}
	}
	// PORT and API_URL must NOT be flagged
	if _, ok := got["PORT"]; ok {
		t.Error("PORT should not be flagged as secret")
	}
	if _, ok := got["API_URL"]; ok {
		t.Error("API_URL should not be flagged as secret")
	}
}

func TestScanSecretsClean(t *testing.T) {
	f := ParseEnv("# required\nAPI_URL=https://example.com\nPORT=8080\nLOG_LEVEL=info\nYOUR_API_KEY=your-api-key-here\nDUMMY_SECRET=xxx\nMAX_TOKENS_PER_REQUEST=500\nTOKEN_BUCKET_SIZE=100\n")
	findings := ScanSecrets(f)
	if len(findings) != 0 {
		t.Errorf("clean example file flagged: %+v", findings)
	}
}

func TestValidateConfig(t *testing.T) {
	cfg := &Config{Rules: map[string]Rule{
		"API_URL": {Required: true, Type: "url"},
		"PORT":    {Required: true, Type: "port"},
		"DEBUG":   {Type: "boolean"},
	}}
	f := ParseEnv("PORT=3000\nDEBUG=yes\n")
	issues := cfg.Validate(f)
	got := map[string]string{}
	for _, i := range issues {
		got[i.Key] = i.Field
	}
	if got["API_URL"] != "required" {
		t.Errorf("API_URL missing should be required issue, got %v", got)
	}
	if _, ok := got["PORT"]; ok {
		t.Errorf("PORT valid, no issue expected: %v", got)
	}
	if _, ok := got["DEBUG"]; ok {
		t.Errorf("DEBUG=yes valid boolean, no issue expected: %v", got)
	}
	f2 := ParseEnv("API_URL=notaurl\nPORT=99999\nDEBUG=maybe\n")
	issues2 := cfg.Validate(f2)
	got2 := map[string]string{}
	for _, i := range issues2 {
		got2[i.Key] = i.Field
	}
	if got2["API_URL"] != "type" {
		t.Errorf("API_URL bad url: want type issue, got %v", got2)
	}
	if got2["PORT"] != "type" {
		t.Errorf("PORT 99999: want type issue, got %v", got2)
	}
	if got2["DEBUG"] != "type" {
		t.Errorf("DEBUG=maybe: want type issue, got %v", got2)
	}
}

func TestRequiredFromComments(t *testing.T) {
	content := "# required\nDATABASE_URL=postgres://x\n" +
		"LOG_LEVEL=info # required\n" +
		"OPTIONAL_VAR=1\n"
	req := requiredFromComments(content)
	if !req["DATABASE_URL"] {
		t.Error("comment above var should mark required")
	}
	if !req["LOG_LEVEL"] {
		t.Error("inline # required should mark required")
	}
	if req["OPTIONAL_VAR"] {
		t.Error("unmarked var should not be required")
	}
}

func TestRenderScan(t *testing.T) {
	f := ParseEnv("TOKEN=ghp_1234567890123456789012345678901234567890\n")
	findings := ScanSecrets(f)
	var buf bytes.Buffer
	renderScan(&buf, f, findings)
	if !strings.Contains(buf.String(), "ghp_") {
		t.Errorf("redacted value should still hint the key name: %s", buf.String())
	}
	if strings.Contains(buf.String(), "ghp_1234567890123456789012345678901234567890") {
		t.Errorf("full value leaked to output: %s", buf.String())
	}
}