package sysproxy

import (
	"net/url"
	"testing"
)

func TestParseProxyServerPlain(t *testing.T) {
	u := parseProxyServer("127.0.0.1:7890", "https")
	if u == nil || u.Scheme != "http" || u.Host != "127.0.0.1:7890" {
		t.Fatalf("got %#v", u)
	}
}

func TestParseProxyServerByScheme(t *testing.T) {
	raw := "http=127.0.0.1:7890;https=127.0.0.1:7891;socks=127.0.0.1:7892"
	u := parseProxyServer(raw, "https")
	if u == nil || u.Host != "127.0.0.1:7891" {
		t.Fatalf("https got %#v", u)
	}
	u = parseProxyServer(raw, "http")
	if u == nil || u.Host != "127.0.0.1:7890" {
		t.Fatalf("http got %#v", u)
	}
}

func TestShouldBypass(t *testing.T) {
	if !shouldBypass("127.0.0.1", "") {
		t.Fatal("loopback should bypass")
	}
	if !shouldBypass("api.spotify.com", "*.spotify.com;<local>") {
		t.Fatal("wildcard bypass")
	}
	if shouldBypass("api.spotify.com", "localhost;<local>") {
		t.Fatal("should not bypass")
	}
	if !shouldBypass("intranet", "<local>") {
		t.Fatal("<local> simple host")
	}
}

func TestIsSelfProxy(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:18513")
	if !isSelfProxy(u, 18513) {
		t.Fatal("expected self")
	}
	if isSelfProxy(u, 7890) {
		t.Fatal("different port")
	}
}
