package rules

import "testing"

func TestHostMatchWildcard(t *testing.T) {
	if !hostMatch("guc3-spclient.spotify.com", "*.spotify.com") {
		t.Fatal("expected subdomain match")
	}
	if hostMatch("evil-spotify.com", "*.spotify.com") {
		t.Fatal("should not match lookalike")
	}
	if !hostMatch("guc3-spclient.spotify.com", "guc3-spclient.spotify.com") {
		t.Fatal("exact match")
	}
}

func TestLoadDefault(t *testing.T) {
	r, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsBlocked("guc3-spclient.spotify.com", "/ad-logic/prefetch") {
		t.Fatal("should block ad-logic")
	}
	if r.IsBlocked("guc3-spclient.spotify.com", "/gabo-receiver-service/v3/events") {
		t.Fatal("should not block gabo")
	}
}
