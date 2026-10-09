//go:build windows

package ui

import (
	"golang.org/x/sys/windows"
)

const trayMutexName = "Local\\SpotifyAdFree_Tray"

var trayMutex windows.Handle

// AcquireTraySingleton 确保托盘只运行一份。返回 false 表示已有实例。
func AcquireTraySingleton() (bool, error) {
	name, err := windows.UTF16PtrFromString(trayMutexName)
	if err != nil {
		return false, err
	}
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	trayMutex = h
	return true, nil
}
