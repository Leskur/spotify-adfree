//go:build windows

package ui

import "os/exec"

func OpenURL(raw string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", raw).Start()
}
