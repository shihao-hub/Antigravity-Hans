//go:build gui && !windows

package main

import (
	"fmt"
	"os"
)

// GUI 壳仅支持 Windows；以 gui 标签在其他平台构建时给出明确提示。
func main() {
	fmt.Fprintln(os.Stderr, "Antigravity-Hans GUI 模式目前仅支持 Windows。")
	os.Exit(1)
}
