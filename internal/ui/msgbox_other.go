//go:build !windows

package ui

import "fmt"

func ShowError(title, message string) {
	fmt.Printf("%s: %s\n", title, message)
}

func ShowInfo(title, message string) {
	fmt.Printf("%s: %s\n", title, message)
}
