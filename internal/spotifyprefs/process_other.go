//go:build !windows

package spotifyprefs

import "fmt"

func Restart() error {
	return fmt.Errorf("仅支持 Windows")
}

func RestartIfRunning() error {
	return nil
}

func QuitIfRunning() error {
	return nil
}

func StartIfNotRunning() error {
	return nil
}
