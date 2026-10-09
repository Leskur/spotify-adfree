package spotifyprefs

import "testing"

func TestUpsertAndRemoveProxy(t *testing.T) {
	in := "ui.theme=dark\nnetwork.proxy.mode=1\nfoo=bar\n"
	out := upsertProxy(in, "2", "127.0.0.1:18513@http")
	if !containsLine(out, `network.proxy.mode=2`) {
		t.Fatalf("mode: %q", out)
	}
	if !containsLine(out, `network.proxy.addr="127.0.0.1:18513@http"`) {
		t.Fatalf("addr: %q", out)
	}
	cleared := removeProxy(out)
	if containsLine(cleared, "network.proxy.mode") || containsLine(cleared, "network.proxy.addr") {
		t.Fatalf("still has proxy: %q", cleared)
	}
	if !containsLine(cleared, "ui.theme=dark") || !containsLine(cleared, "foo=bar") {
		t.Fatalf("lost other keys: %q", cleared)
	}
}

func containsLine(s, want string) bool {
	for _, line := range splitKeepEnd(s) {
		if line == want {
			return true
		}
	}
	return false
}
