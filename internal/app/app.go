package app

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/leskur/spotify-ad-free/internal/rewrite"
	"github.com/leskur/spotify-ad-free/internal/rules"
	"github.com/leskur/spotify-ad-free/internal/spotifyprefs"
)

type App struct {
	Rules   *rules.Rules
	Rewrite *rewrite.Server
	ca      *rewrite.CA
	paths   spotifyprefs.Paths
}

func New() (*App, error) {
	state, err := stateDir()
	if err != nil {
		return nil, err
	}
	r, err := rules.LoadDefault()
	if err != nil {
		return nil, err
	}

	ca, err := rewrite.LoadOrCreateCA(filepath.Join(state, "certs"))
	if err != nil {
		return nil, err
	}

	paths, err := spotifyprefs.ResolvePaths(state)
	if err != nil {
		return nil, err
	}

	return &App{Rules: r, ca: ca, paths: paths}, nil
}

func stateDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Spotify AdFree")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) Start() error {
	if err := a.ca.InstallRoot(); err != nil {
		return err
	}

	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", rewrite.DefaultPort))
	rw := rewrite.NewServer(addr, a.Rules, a.ca)
	rw.SimplifyLyrics.Store(a.paths.LoadLyricsSimplified())
	if err := rw.Start(); err != nil {
		return fmt.Errorf("无法监听代理端口")
	}
	a.Rewrite = rw
	return nil
}

// AfterStart 启动本地代理后：按偏好写入托管代理，并可选拉起 Spotify。
func (a *App) AfterStart() error {
	if a.paths.LoadManaged() {
		if err := a.paths.Apply(a.ProxyPort()); err != nil {
			return err
		}
		if err := spotifyprefs.RestartIfRunning(); err != nil {
			return err
		}
	}
	if a.paths.LoadStartSpotifyOnLaunch() {
		return spotifyprefs.StartIfNotRunning()
	}
	return nil
}

// SpotifyProxyManaged 托管偏好是否开启。
func (a *App) SpotifyProxyManaged() bool {
	return a.paths.LoadManaged()
}

// EnableSpotifyProxy 勾选：写入代理并重启。
func (a *App) EnableSpotifyProxy() error {
	if err := a.paths.Apply(a.ProxyPort()); err != nil {
		return err
	}
	if err := a.paths.SaveManaged(true); err != nil {
		return err
	}
	return spotifyprefs.RestartIfRunning()
}

// DisableSpotifyProxy 取消勾选：还原并重启。
func (a *App) DisableSpotifyProxy() error {
	if err := a.paths.Restore(a.ProxyPort()); err != nil {
		return err
	}
	if err := a.paths.SaveManaged(false); err != nil {
		return err
	}
	return spotifyprefs.RestartIfRunning()
}

func (a *App) QuitSpotifyOnExit() bool {
	return a.paths.LoadQuitSpotifyOnExit()
}

func (a *App) SetQuitSpotifyOnExit(on bool) error {
	return a.paths.SaveQuitSpotifyOnExit(on)
}

func (a *App) StartSpotifyOnLaunch() bool {
	return a.paths.LoadStartSpotifyOnLaunch()
}

func (a *App) SetStartSpotifyOnLaunch(on bool) error {
	return a.paths.SaveStartSpotifyOnLaunch(on)
}

func (a *App) LyricsSimplified() bool {
	return a.paths.LoadLyricsSimplified()
}

func (a *App) SetLyricsSimplified(on bool) error {
	if err := a.paths.SaveLyricsSimplified(on); err != nil {
		return err
	}
	if a.Rewrite != nil {
		a.Rewrite.SimplifyLyrics.Store(on)
	}
	return nil
}

func (a *App) Stop() error {
	// 退出前若 prefs 仍指向我们，必须先还原成功再动 Spotify / 关代理
	port := a.ProxyPort()
	if a.paths.IsOurProxy(port) {
		if err := a.paths.Restore(port); err != nil {
			return fmt.Errorf("还原 Spotify 代理失败")
		}
		if a.paths.LoadQuitSpotifyOnExit() {
			_ = spotifyprefs.QuitIfRunning()
		} else {
			_ = spotifyprefs.RestartIfRunning()
		}
	}
	if a.Rewrite == nil {
		return nil
	}
	err := a.Rewrite.Stop()
	a.Rewrite = nil
	return err
}

func (a *App) ProxyPort() int {
	if a.Rewrite != nil {
		return a.Rewrite.Port()
	}
	return rewrite.DefaultPort
}

func (a *App) ProxyAddr() string {
	return net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", a.ProxyPort()))
}
