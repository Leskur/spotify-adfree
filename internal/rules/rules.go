package rules

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type RewriteRule struct {
	Match        string `yaml:"match"`
	RedirectHost string `yaml:"redirect_host"`

	Host string `yaml:"-"`
	Path string `yaml:"-"`
}

type URLPattern struct {
	Host string
	Path string
}

type Rules struct {
	Rewrites []RewriteRule `yaml:"rewrites"`
	Blocks   []string      `yaml:"blocks"`

	blockPatterns []URLPattern `yaml:"-"`
}

func Parse(data []byte) (*Rules, error) {
	var r Rules
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	if len(r.Rewrites) == 0 && len(r.Blocks) == 0 {
		return nil, fmt.Errorf("rules: rewrites/blocks is empty")
	}
	for i := range r.Rewrites {
		if err := r.Rewrites[i].normalize(); err != nil {
			return nil, err
		}
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

func (rw *RewriteRule) normalize() error {
	u, err := url.Parse(strings.TrimSpace(rw.Match))
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("invalid rewrite match: %q", rw.Match)
	}
	if strings.TrimSpace(rw.RedirectHost) == "" {
		return fmt.Errorf("rewrite redirect_host is empty for %q", rw.Match)
	}
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	rw.Host = strings.ToLower(u.Hostname())
	rw.Path = p
	rw.RedirectHost = strings.ToLower(strings.TrimSpace(rw.RedirectHost))
	return nil
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

func normalizeHost(host string) string {
	host = strings.ToLower(host)
	if h, _, ok := strings.Cut(host, ":"); ok {
		host = h
	}
	return host
}

func (r *Rules) RedirectHost(host, path string) string {
	host = normalizeHost(host)
	for _, rw := range r.Rewrites {
		if rw.Host == host && pathMatch(path, rw.Path) {
			return rw.RedirectHost
		}
	}
	return ""
}

func (r *Rules) IsBlocked(host, path string) bool {
	host = normalizeHost(host)
	for _, b := range r.blockPatterns {
		if b.Host == host && pathMatch(path, b.Path) {
			return true
		}
	}
	return false
}

func Load(path string) (*Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules: %w", err)
	}
	return Parse(data)
}

func ResolvePath(stateDir string) (string, error) {
	if p := os.Getenv("SPOTIFY_AD_FREE_RULES"); p != "" {
		return p, nil
	}

	exe, _ := os.Executable()
	exeDir := "."
	if exe != "" {
		exeDir = filepath.Dir(exe)
	}

	candidates := []string{
		filepath.Join("configs", "rules.yaml"),
		filepath.Join(exeDir, "configs", "rules.yaml"),
		filepath.Join(stateDir, "rules.yaml"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, err := filepath.Abs(c)
			if err != nil {
				return c, nil
			}
			return abs, nil
		}
	}

	dest := filepath.Join(stateDir, "rules.yaml")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, defaultRulesYAML, 0o644); err != nil {
		return "", err
	}
	return dest, nil
}
