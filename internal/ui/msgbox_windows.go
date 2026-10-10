//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

func ShowError(title, message string) {
	messageBox(title, message, 0x10) // MB_ICONERROR
}

func ShowInfo(title, message string) {
	messageBox(title, message, 0x40) // MB_ICONINFORMATION
}

// AskYesNo 返回用户是否点了「是」。
func AskYesNo(title, message string) bool {
	const mbYesNo = 0x04
	const mbIconQuestion = 0x20
	const idYes = 6
	r := messageBox(title, message, mbYesNo|mbIconQuestion)
	return r == idYes
}

func messageBox(title, message string, flags uintptr) int {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	r, _, _ := messageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
	return int(r)
}
