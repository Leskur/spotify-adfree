package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/leskur/spotify-ad-free/internal/sysproxy"
	"github.com/leskur/spotify-ad-free/internal/version"
)

type Result struct {
	Current string
	Latest  string
	Newer   bool
	URL     string
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// Check 查询 GitHub latest release，与当前版本比较。
func Check() (*Result, error) {
	api := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/releases/latest",
		version.GitHubOwner, version.GitHubRepo,
	)
	req, err := http.NewRequest(http.MethodGet, api, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", version.AppName+"/"+version.Version)

	client := &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				return sysproxy.HTTP(r, 0)
			},
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接更新服务器")
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("尚未发布版本")
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("检查更新失败")
	}

	var rel githubRelease
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("检查更新失败")
	}
	latest := strings.TrimSpace(rel.TagName)
	if latest == "" {
		return nil, fmt.Errorf("尚未发布版本")
	}
	page := strings.TrimSpace(rel.HTMLURL)
	if page == "" {
		page = fmt.Sprintf("https://github.com/%s/%s/releases/latest", version.GitHubOwner, version.GitHubRepo)
	}

	newer, err := isNewer(latest, version.Version)
	if err != nil {
		return nil, err
	}
	return &Result{
		Current: version.Version,
		Latest:  trimV(latest),
		Newer:   newer,
		URL:     page,
	}, nil
}

func trimV(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		return s[1:]
	}
	return s
}

func isNewer(latest, current string) (bool, error) {
	lp, err := parseSemver(latest)
	if err != nil {
		return false, err
	}
	cp, err := parseSemver(current)
	if err != nil {
		return false, err
	}
	for i := 0; i < 3; i++ {
		if lp[i] > cp[i] {
			return true, nil
		}
		if lp[i] < cp[i] {
			return false, nil
		}
	}
	return false, nil
}

func parseSemver(s string) ([3]int, error) {
	var out [3]int
	s = trimV(s)
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return out, fmt.Errorf("无效版本: %s", s)
	}
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 {
			return out, fmt.Errorf("无效版本: %s", s)
		}
		out[i] = n
	}
	return out, nil
}
