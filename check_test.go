package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	ref := ParseEnv("A=1\nB=2\nC=3\nD=\n")
	act := ParseEnv("B=2\nC=\nE=9\n")
	issues := Diff(ref, act)
	got := map[string]Status{}
	for _, i := range issues {
		got[i.Key] = i.Status
	}
	want := map[string]Status{
		"A": StatusMissing,
		"C": StatusEmpty,
		"D": StatusMissing,
		"E": StatusExtra,
	}
	if len(got) != len(want) {
		t.Fatalf("want %d issues, got %d: %v", len(want), len(got), got)
	}
	for k, s := range want {
		if got[k] != s {
			t.Errorf("key %s: want status %v, got %v", k, s, got[k])
		}
	}
}

func TestCompareTable(t *testing.T) {
	a := ParseEnv("A=1\nB=2\n")
	a.Name = ".env.local"
	b := ParseEnv("A=1\nC=\n")
	b.Name = ".env.staging"
	var buf bytes.Buffer
	issues := CompareTable(&buf, []*EnvFile{a, b})
	// B missing in staging, C missing in local, C empty in staging.
	if issues != 3 {
		t.Errorf("want 3 issues, got %d", issues)
	}
	if !strings.Contains(buf.String(), "MISSING") || !strings.Contains(buf.String(), "EMPTY") {
		t.Errorf("table should contain MISSING and EMPTY cells, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), ".env.local") || !strings.Contains(buf.String(), ".env.staging") {
		t.Errorf("table should contain file names, got:\n%s", buf.String())
	}
}