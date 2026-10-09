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

func messageBox(title, message string, flags uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	messageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
}
