//go:build windows

package spotifyprefs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const spotifyExeName = "Spotify.exe"

// Restart 结束正在运行的 Spotify 并重新启动；未运行则直接启动。
func Restart() error {
	exe, running := findSpotify()
	if running {
		if err := killSpotify(); err != nil {
			return err
		}
		if err := waitSpotifyGone(5 * time.Second); err != nil {
			return err
		}
	}
	if exe == "" {
		var err error
		exe, err = locateSpotifyExe()
		if err != nil {
			return err
		}
	}
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	// 独立进程启动 GUI，不要 HideWindow
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS
	return cmd.Start()
}

// RestartIfRunning 仅当 Spotify 在跑时重启。
func RestartIfRunning() error {
	_, running := findSpotify()
	if !running {
		return nil
	}
	return Restart()
}

// QuitIfRunning 仅结束 Spotify，不重新启动。
func QuitIfRunning() error {
	_, running := findSpotify()
	if !running {
		return nil
	}
	if err := killSpotify(); err != nil {
		return err
	}
	return waitSpotifyGone(5 * time.Second)
}

func waitSpotifyGone(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, running := findSpotify(); !running {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, running := findSpotify(); running {
		return fmt.Errorf("Spotify 未能及时退出")
	}
	return nil
}

// StartIfNotRunning 未运行时启动 Spotify；已在跑则不动。
func StartIfNotRunning() error {
	exe, running := findSpotify()
	if running {
		return nil
	}
	if exe == "" {
		var err error
		exe, err = locateSpotifyExe()
		if err != nil {
			return err
		}
	}
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008}
	return cmd.Start()
}

func locateSpotifyExe() (string, error) {
	roaming, err := os.UserConfigDir()
	if err == nil {
		p := filepath.Join(roaming, "Spotify", spotifyExeName)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	local := os.Getenv("LOCALAPPDATA")
	if local != "" {
		p := filepath.Join(local, "Spotify", spotifyExeName)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("未找到 Spotify")
}

func findSpotify() (exe string, running bool) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(snap)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return "", false
	}
	for {
		name := windows.UTF16ToString(pe.ExeFile[:])
		if equalFold(name, spotifyExeName) {
			running = true
			if path, err := processImagePath(pe.ProcessID); err == nil {
				exe = path
			}
			// 继续扫，优先拿到带路径的
			if exe != "" {
				return exe, true
			}
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return exe, running
}

func processImagePath(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

func killSpotify() error {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snap)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return err
	}
	var last error
	for {
		name := windows.UTF16ToString(pe.ExeFile[:])
		if equalFold(name, spotifyExeName) {
			h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pe.ProcessID)
			if err != nil {
				last = err
			} else {
				err = windows.TerminateProcess(h, 1)
				windows.CloseHandle(h)
				if err != nil {
					last = err
				}
			}
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return last
}

func equalFold(a, b string) bool {
	return strings.EqualFold(a, b)
}
