package ui

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/energye/systray"

	"github.com/leskur/spotify-ad-free/internal/app"
	"github.com/leskur/spotify-ad-free/internal/update"
	"github.com/leskur/spotify-ad-free/internal/version"
)

//go:embed icons/tray.ico
var trayIcon []byte

// RunTray 托盘模式：运行即开启本地代理。
func RunTray() {
	HideConsole()

	ok, err := AcquireTraySingleton()
	if err != nil {
		ShowError(version.AppName, "无法启动")
		os.Exit(1)
	}
	if !ok {
		ShowInfo(version.AppName, "程序已在运行中。\n请看右下角托盘图标。")
		os.Exit(0)
	}

	systray.Run(onReady, func() {})
}

func onReady() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip(version.AppName)
	systray.SetOnRClick(func(menu systray.IMenu) { _ = menu.ShowMenu() })
	systray.SetOnClick(func(menu systray.IMenu) { _ = menu.ShowMenu() })

	mBlockAds := systray.AddMenuItem("屏蔽广告", "")
	mBlockAds.Disable()
	mLyrics := systray.AddMenuItem("歌词", "")
	mLyricsHans := mLyrics.AddSubMenuItem("简体中文", "")
	mSettings := systray.AddMenuItem("设置", "")
	mStartSpotify := mSettings.AddSubMenuItem("启动时打开 Spotify", "")
	mQuitSpotify := mSettings.AddSubMenuItem("退出时关闭 Spotify", "")
	mAbout := systray.AddMenuItem("关于", "")
	mCheckUpdate := mAbout.AddSubMenuItem("检查更新", "")
	mVer := mAbout.AddSubMenuItem("版本 v"+version.Version, "")
	mVer.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "")

	a, err := app.New()
	if err != nil {
		ShowError(version.AppName, "启动失败")
		systray.Quit()
		return
	}

	syncChecks := func() {
		if a.SpotifyProxyManaged() {
			mBlockAds.Check()
		} else {
			mBlockAds.Uncheck()
		}
		if a.LyricsSimplified() {
			mLyricsHans.Check()
		} else {
			mLyricsHans.Uncheck()
		}
		if a.StartSpotifyOnLaunch() {
			mStartSpotify.Check()
		} else {
			mStartSpotify.Uncheck()
		}
		if a.QuitSpotifyOnExit() {
			mQuitSpotify.Check()
		} else {
			mQuitSpotify.Uncheck()
		}
	}

	mBlockAds.Click(func() {
		var err error
		if a.SpotifyProxyManaged() {
			err = a.DisableSpotifyProxy()
		} else {
			err = a.EnableSpotifyProxy()
		}
		if err != nil {
			ShowError(version.AppName, err.Error())
		}
		syncChecks()
	})
	mLyricsHans.Click(func() {
		next := !a.LyricsSimplified()
		if err := a.SetLyricsSimplified(next); err != nil {
			ShowError(version.AppName, "保存设置失败")
		}
		syncChecks()
	})
	mStartSpotify.Click(func() {
		next := !a.StartSpotifyOnLaunch()
		if err := a.SetStartSpotifyOnLaunch(next); err != nil {
			ShowError(version.AppName, "保存设置失败")
		}
		syncChecks()
	})
	mQuitSpotify.Click(func() {
		next := !a.QuitSpotifyOnExit()
		if err := a.SetQuitSpotifyOnExit(next); err != nil {
			ShowError(version.AppName, "保存设置失败")
		}
		syncChecks()
	})
	mCheckUpdate.Click(func() {
		mCheckUpdate.Disable()
		go func() {
			defer mCheckUpdate.Enable()
			res, err := update.Check()
			if err != nil {
				ShowError(version.AppName, err.Error())
				return
			}
			if !res.Newer {
				ShowInfo(version.AppName, "当前已是最新版本。\n版本 v"+res.Current)
				return
			}
			msg := fmt.Sprintf("发现新版本 v%s（当前 v%s）。\n是否打开下载页面？", res.Latest, res.Current)
			if AskYesNo(version.AppName, msg) {
				if err := OpenURL(res.URL); err != nil {
					ShowError(version.AppName, "无法打开下载页面")
				}
			}
		}()
	})
	mQuit.Click(func() {
		if err := a.Stop(); err != nil {
			ShowError(version.AppName, err.Error())
			return
		}
		systray.Quit()
	})

	go func() {
		time.Sleep(200 * time.Millisecond)
		if err := a.Start(); err != nil {
			msg := err.Error()
			if msg == "" {
				msg = "启动失败"
			}
			ShowError(version.AppName, msg)
			systray.Quit()
			return
		}
		mBlockAds.Enable()

		if err := a.AfterStart(); err != nil {
			ShowError(version.AppName, err.Error())
		}
		syncChecks()
	}()
}
