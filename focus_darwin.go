//go:build darwin

package main

import (
	"os/exec"
	"path/filepath"
)

// Repeated clicks should focus an existing normal-app window, without racing
// the monitor by launching a second instance before the debug flags are ready.
func focusRunningNormalApp(cfg AppConfig) {
	if cfg.Name != AppNormal.Name {
		return
	}
	app := DetectApp(cfg)
	if !app.Running || !isExpectedExecutable(app.Path, cfg) {
		return
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(app.Path)))
	_ = exec.Command("/usr/bin/open", "-a", bundle).Run()
}
