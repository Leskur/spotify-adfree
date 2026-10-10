package spotifyprefs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	proxyModeKey = "network.proxy.mode"
	proxyAddrKey = "network.proxy.addr"
)

// Paths 返回 Spotify prefs 与我们的状态文件路径。
type Paths struct {
	Prefs  string
	Backup string
}

func ResolvePaths(stateDir string) (Paths, error) {
	roaming, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		Prefs:  filepath.Join(roaming, "Spotify", "prefs"),
		Backup: filepath.Join(stateDir, "spotify-prefs.bak"),
	}, nil
}

func OurAddr(port int) string {
	return fmt.Sprintf("127.0.0.1:%d@http", port)
}

func (p Paths) IsOurProxy(port int) bool {
	mode, addr, err := p.readProxy()
	if err != nil {
		return false
	}
	want := OurAddr(port)
	return mode == "2" && normalizeAddr(addr) == normalizeAddr(want)
}

func normalizeAddr(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	return strings.ToLower(s)
}

func (p Paths) readProxy() (mode, addr string, err error) {
	data, err := os.ReadFile(p.Prefs)
	if err != nil {
		return "", "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case proxyModeKey:
			mode = strings.Trim(val, `"`)
		case proxyAddrKey:
			addr = val
		}
	}
	return mode, addr, nil
}

// Apply 备份（若尚无）并写入我们的代理。
func (p Paths) Apply(port int) error {
	if _, err := os.Stat(p.Prefs); err != nil {
		return fmt.Errorf("未找到 Spotify 配置")
	}
	if !p.IsOurProxy(port) {
		if _, err := os.Stat(p.Backup); os.IsNotExist(err) {
			data, err := os.ReadFile(p.Prefs)
			if err != nil {
				return err
			}
			if err := os.WriteFile(p.Backup, data, 0o644); err != nil {
				return err
			}
		}
	}
	data, err := os.ReadFile(p.Prefs)
	if err != nil {
		return err
	}
	next := upsertProxy(string(data), "2", OurAddr(port))
	return os.WriteFile(p.Prefs, []byte(next), 0o644)
}

// Restore 从备份恢复；无备份则去掉我们的代理行。
func (p Paths) Restore(port int) error {
	if _, err := os.Stat(p.Backup); err == nil {
		data, err := os.ReadFile(p.Backup)
		if err != nil {
			return err
		}
		if err := os.WriteFile(p.Prefs, data, 0o644); err != nil {
			return err
		}
		_ = os.Remove(p.Backup)
		return nil
	}
	if !p.IsOurProxy(port) {
		return nil
	}
	data, err := os.ReadFile(p.Prefs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	next := removeProxy(string(data))
	return os.WriteFile(p.Prefs, []byte(next), 0o644)
}

func upsertProxy(content, mode, addr string) string {
	lines := splitKeepEnd(content)
	var out []string
	sawMode, sawAddr := false, false
	modeLine := proxyModeKey + "=" + mode
	addrLine := proxyAddrKey + "=" + `"` + addr + `"`
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			out = append(out, line)
			continue
		}
		key, _, ok := strings.Cut(trim, "=")
		if !ok {
			out = append(out, line)
			continue
		}
		key = strings.TrimSpace(key)
		switch key {
		case proxyModeKey:
			out = append(out, modeLine)
			sawMode = true
		case proxyAddrKey:
			out = append(out, addrLine)
			sawAddr = true
		default:
			out = append(out, line)
		}
	}
	if !sawMode {
		out = append(out, modeLine)
	}
	if !sawAddr {
		out = append(out, addrLine)
	}
	return joinLines(out)
}

func removeProxy(content string) string {
	lines := splitKeepEnd(content)
	var out []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		key, _, ok := strings.Cut(trim, "=")
		if ok {
			key = strings.TrimSpace(key)
			if key == proxyModeKey || key == proxyAddrKey {
				continue
			}
		}
		out = append(out, line)
	}
	return joinLines(out)
}

func splitKeepEnd(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.Split(content, "\n")
}

func joinLines(lines []string) string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
