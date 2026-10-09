//go:build !windows

package ui

func AcquireTraySingleton() (bool, error) {
	return true, nil
}
