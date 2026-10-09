//go:build windows

package sysproxy

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func systemHTTP(req *http.Request, listenPort int) (*url.URL, error) {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return nil, nil
	}
	defer k.Close()

	enable, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || enable == 0 {
		return nil, nil
	}

	server, _, err := k.GetStringValue("ProxyServer")
	if err != nil || strings.TrimSpace(server) == "" {
		return nil, nil
	}

	host := req.URL.Hostname()
	if host == "" {
		host = req.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
	}
	override, _, _ := k.GetStringValue("ProxyOverride")
	if shouldBypass(host, override) {
		return nil, nil
	}

	u := parseProxyServer(server, req.URL.Scheme)
	if u == nil || isSelfProxy(u, listenPort) {
		return nil, nil
	}
	return u, nil
}
