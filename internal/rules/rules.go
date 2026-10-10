package rules

import (
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

type URLPattern struct {
	Host string
	Path string
}

type Rules struct {
	Blocks []string `yaml:"blocks"`

	blockPatterns []URLPattern `yaml:"-"`
}

func Parse(data []byte) (*Rules, error) {
	var r Rules
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	if len(r.Blocks) == 0 {
		return nil, fmt.Errorf("rules: blocks is empty")
	}
	for _, raw := range r.Blocks {
		p, err := parseURLPattern(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid block: %w", err)
		}
		r.blockPatterns = append(r.blockPatterns, p)
	}
	return &r, nil
}

func LoadDefault() (*Rules, error) {
	return Parse(defaultRulesYAML)
}

func parseURLPattern(raw string) (URLPattern, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return URLPattern{}, fmt.Errorf("invalid url: %q", raw)
	}
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	return URLPattern{
		Host: strings.ToLower(u.Hostname()),
		Path: p,
	}, nil
}

func pathMatch(got, want string) bool {
	if got == "" {
		got = "/"
	}
	return got == want || strings.HasPrefix(got, want+"/")
}

// hostMatch 支持精确主机名，或 "*.example.com" 匹配其子域。
func hostMatch(got, want string) bool {
	if got == want {
		return true
	}
	if strings.HasPrefix(want, "*.") {
		suf := want[1:] // ".example.com"
		return strings.HasSuffix(got, suf)
	}
	return false
}

func normalizeHost(host string) string {
	host = strings.ToLower(host)
	if h, _, ok := strings.Cut(host, ":"); ok {
		host = h
	}
	return host
}

func (r *Rules) IsBlocked(host, path string) bool {
	host = normalizeHost(host)
	for _, b := range r.blockPatterns {
		if hostMatch(host, b.Host) && pathMatch(path, b.Path) {
			return true
		}
	}
	return false
}
