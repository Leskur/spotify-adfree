package sysproxy

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// HTTP 按当前系统代理返回上游代理 URL。
// 未启用、命中绕过列表、或代理指向本程序 listenPort 时返回 nil（直连）。
func HTTP(req *http.Request, listenPort int) (*url.URL, error) {
	if req == nil || req.URL == nil {
		return nil, nil
	}
	return systemHTTP(req, listenPort)
}

func parseProxyServer(server, scheme string) *url.URL {
	server = strings.TrimSpace(server)
	if server == "" {
		return nil
	}

	// "host:port" 或 "http=h:p;https=h:p;socks=h:p"
	if !strings.Contains(server, "=") {
		return proxyURL("http", server)
	}

	parts := strings.Split(server, ";")
	byScheme := map[string]string{}
	var first string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		byScheme[k] = v
		if first == "" {
			first = v
		}
	}

	scheme = strings.ToLower(scheme)
	if scheme == "https" {
		if v := byScheme["https"]; v != "" {
			return proxyURL("http", v)
		}
		if v := byScheme["http"]; v != "" {
			return proxyURL("http", v)
		}
	}
	if scheme == "http" {
		if v := byScheme["http"]; v != "" {
			return proxyURL("http", v)
		}
	}
	if v := byScheme["socks"]; v != "" {
		return proxyURL("socks5", v)
	}
	if v := byScheme["socks5"]; v != "" {
		return proxyURL("socks5", v)
	}
	if first != "" {
		return proxyURL("http", first)
	}
	return nil
}

func proxyURL(scheme, hostPort string) *url.URL {
	hostPort = strings.TrimSpace(hostPort)
	hostPort = strings.TrimPrefix(hostPort, "http://")
	hostPort = strings.TrimPrefix(hostPort, "https://")
	hostPort = strings.TrimPrefix(hostPort, "socks5://")
	hostPort = strings.TrimPrefix(hostPort, "socks://")
	if hostPort == "" {
		return nil
	}
	if _, _, err := net.SplitHostPort(hostPort); err != nil {
		// 可能只有 host 无端口
		hostPort = net.JoinHostPort(hostPort, "80")
	}
	return &url.URL{Scheme: scheme, Host: hostPort}
}

func shouldBypass(host, override string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}

	override = strings.TrimSpace(override)
	if override == "" {
		return false
	}
	for _, raw := range strings.Split(override, ";") {
		pat := strings.ToLower(strings.TrimSpace(raw))
		if pat == "" {
			continue
		}
		if pat == "<local>" {
			if !strings.Contains(host, ".") {
				return true
			}
			continue
		}
		if matchHostPattern(host, pat) {
			return true
		}
	}
	return false
}

func matchHostPattern(host, pat string) bool {
	if pat == host {
		return true
	}
	if strings.HasPrefix(pat, "*.") {
		suf := pat[1:] // ".example.com"
		return strings.HasSuffix(host, suf) || host == strings.TrimPrefix(pat, "*.")
	}
	if strings.HasPrefix(pat, "*") {
		return strings.HasSuffix(host, strings.TrimPrefix(pat, "*"))
	}
	if strings.HasSuffix(pat, "*") {
		return strings.HasPrefix(host, strings.TrimSuffix(pat, "*"))
	}
	return false
}

func isSelfProxy(u *url.URL, listenPort int) bool {
	if u == nil || listenPort <= 0 {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return false
	}
	p := u.Port()
	if p == "" {
		return false
	}
	n, err := strconv.Atoi(p)
	return err == nil && n == listenPort
}
