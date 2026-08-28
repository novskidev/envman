package main

import "testing"

func TestParseRemoteTarget(t *testing.T) {
	tr, err := ParseRemoteTarget("deploy@vps.example.com:/srv/app/.env", "local", 0)
	if err != nil {
		t.Fatal(err)
	}
	if tr.User != "deploy" || tr.Host != "vps.example.com" || tr.Port != 22 || tr.Path != "/srv/app/.env" {
		t.Errorf("bad parse: %+v", tr)
	}

	tr2, err := ParseRemoteTarget("vps.example.com:2222:/srv/.env", "root", 0)
	if err != nil {
		t.Fatal(err)
	}
	if tr2.User != "root" || tr2.Host != "vps.example.com" || tr2.Port != 2222 || tr2.Path != "/srv/.env" {
		t.Errorf("bad parse: %+v", tr2)
	}

	if _, err := ParseRemoteTarget("nonsense", "root", 0); err == nil {
		t.Error("want error for target without path")
	}
	if _, err := ParseRemoteTarget("u@h:", "root", 0); err == nil {
		t.Error("want error for empty path")
	}
	if _, err := ParseRemoteTarget("u@h:99999:/x", "root", 0); err == nil {
		t.Error("want error for bad port")
	}
}