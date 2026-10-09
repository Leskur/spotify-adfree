//go:build windows

package ui

import "syscall"

func HideConsole() {
	getConsole := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
	showWindow := syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
	hwnd, _, _ := getConsole.Call()
	if hwnd == 0 {
		return
	}
	showWindow.Call(hwnd, 0) // SW_HIDE
}
