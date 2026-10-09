//go:build !windows

package rewrite

import "fmt"

func (c *CA) IsRootInstalled() bool { return false }

func (c *CA) InstallRoot() error {
	if c.IsRootInstalled() {
		return nil
	}
	return fmt.Errorf("CA install not implemented on this OS; trust %s manually", c.CertPath())
}
