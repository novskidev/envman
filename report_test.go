package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildReport(t *testing.T) {
	ref := ParseEnv("A=1\nB=2\n")
	act := ParseEnv("A=1\nC=3\n")
	issues := Diff(ref, act) // B missing, C extra

	r := buildReport(ref, act, issues)
	if r.Missing != 1 || r.Extra != 1 || r.Empty != 0 || r.OK != 1 {
		t.Fatalf("counts wrong: %+v", r)
	}
	if r.ExitCode != 1 {
		t.Fatalf("exit code = %d, want 1", r.ExitCode)
	}

	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"missing": 1`) || !strings.Contains(out, `"key": "B"`) {
		t.Fatalf("json missing fields: %s", out)
	}

	buf.Reset()
	r.WriteMarkdown(&buf)
	if !strings.Contains(buf.String(), "MISSING") || !strings.Contains(buf.String(), "1") {
		t.Fatalf("markdown output wrong: %s", buf.String())
	}

	// clean report exits 0
	ok := buildReport(ref, ref, nil)
	if ok.ExitCode != 0 {
		t.Fatalf("clean exit = %d, want 0", ok.ExitCode)
	}
}