//go:build windows

package rewrite

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func (c *CA) thumbprintSHA1() string {
	sum := sha1.Sum(c.Cert.Raw)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func (c *CA) thumbprintSpaced() string {
	tp := c.thumbprintSHA1()
	parts := make([]string, 0, len(tp)/2)
	for i := 0; i+1 < len(tp); i += 2 {
		parts = append(parts, tp[i:i+2])
	}
	return strings.Join(parts, " ")
}

func (c *CA) IsRootInstalled() bool {
	cmd := exec.Command("certutil", "-user", "-store", "Root")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	text := strings.ToUpper(string(out))
	// 只认当前证书指纹，避免同名旧 CA 导致跳过安装新证书
	return strings.Contains(text, c.thumbprintSpaced()) ||
		strings.Contains(text, c.thumbprintSHA1())
}

// InstallRoot 尚未安装时才写入用户信任根。
// 用户在系统提示里点「否」或安装未成功时返回错误，调用方应停止启动。
func (c *CA) InstallRoot() error {
	if c.IsRootInstalled() {
		return nil
	}
	cmd := exec.Command("certutil", "-user", "-addstore", "Root", c.CertPath())
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	_, err := cmd.CombinedOutput()
	if err != nil || !c.IsRootInstalled() {
		// 不附带 certutil 原文：中文 Windows 下多为 GBK，塞进 MessageBox 会乱码
		return fmt.Errorf("未安装本地证书，无法启动")
	}
	return nil
}
