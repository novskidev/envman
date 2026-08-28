package main

import "testing"

func TestParseEnv(t *testing.T) {
	in := "# comment\n" +
		"DATABASE_URL=postgres://localhost/db\n" +
		"JWT_SECRET=\"s3cr3t\"\n" +
		"PORT=3000\n" +
		"EMPTY_VAR=\n" +
		"FLAG='yes'\n" +
		"export EXPORTED=1\n" +
		"NO_EQUALS_HERE\n" +
		"FOO=a=b=c\n"
	f := ParseEnv(in)
	want := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"JWT_SECRET":   "s3cr3t",
		"PORT":         "3000",
		"EMPTY_VAR":    "",
		"FLAG":         "yes",
		"EXPORTED":     "1",
		"FOO":          "a=b=c",
	}
	if len(f.Vars) != len(want) {
		t.Fatalf("want %d vars, got %d: %v", len(want), len(f.Vars), f.Vars)
	}
	for k, v := range want {
		if got := f.Vars[k]; got != v {
			t.Errorf("key %s: want %q, got %q", k, v, got)
		}
	}
	if _, ok := f.Vars["NO_EQUALS_HERE"]; ok {
		t.Error("malformed line without = should be skipped")
	}
	if len(f.Order) != len(want) {
		t.Errorf("Order should have %d entries, got %d", len(want), len(f.Order))
	}
}

func TestParseEnvDuplicate(t *testing.T) {
	f := ParseEnv("DUP=1\nDUP=2\n")
	if f.Vars["DUP"] != "2" {
		t.Errorf("duplicate key: last value should win, got %q", f.Vars["DUP"])
	}
	if len(f.Order) != 1 {
		t.Errorf("duplicate key: Order should have 1 entry, got %d", len(f.Order))
	}
}