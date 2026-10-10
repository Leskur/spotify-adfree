//go:build !windows

package ui

import "fmt"

func OpenURL(raw string) error {
	fmt.Println(raw)
	return nil
}
