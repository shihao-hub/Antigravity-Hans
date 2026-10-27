//go:build windows

package main

import (
	"fmt"
	"os"
)

// Unix mode bits do not describe Windows ACLs. Use the embedded script unless
// the user explicitly opts into trusting an external executable script.
func verifyOverlayOwner(_ os.FileInfo) error {
	return fmt.Errorf("Windows 外部脚本需要通过 --trust-external-overlay 显式信任")
}
