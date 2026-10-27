package main

import (
	"errors"
	"github.com/shirou/gopsutil/v4/process"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestExternalOverlayReadSafety(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.js")
	data := []byte("window." + overlaySentinel + " = true;")
	if err := os.WriteFile(good, data, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := readExternalOverlay(good, true); err != nil || got != string(data) {
		t.Fatalf("trusted regular script: %q %v", got, err)
	}
	if _, err := readExternalOverlay(good, false); (err != nil) != IsWindows {
		t.Fatalf("automatic trust differs from platform policy: %v", err)
	}
	large := filepath.Join(dir, "large.js")
	if err := os.WriteFile(large, make([]byte, maxExternalOverlaySize+1), 0600); err != nil {
		t.Fatal(err)
	}
	for _, trust := range []bool{false, true} {
		if _, err := readExternalOverlay(large, trust); err == nil {
			t.Fatal("size limit must apply even when explicitly trusted")
		}
		if _, err := readExternalOverlay(dir, trust); err == nil {
			t.Fatal("directory accepted as executable JS")
		}
	}
	link := filepath.Join(dir, "link.js")
	if err := os.Symlink(good, link); err == nil {
		for _, trust := range []bool{false, true} {
			if _, err := readExternalOverlay(link, trust); err == nil {
				t.Fatal("symlink accepted")
			}
		}
	}
}

func TestAtomicShortcutFileDoesNotOverwriteLinkedVictim(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(dir, "launcher")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := writeRootFileAtomic(root, "launcher", []byte("new"), 0755); err == nil {
		t.Fatal("symlink destination accepted")
	}
	got, err := os.ReadFile(victim)
	if err != nil || string(got) != "keep" {
		t.Fatalf("victim altered: %q %v", got, err)
	}
	if err := os.Symlink(filepath.Dir(victim), filepath.Join(dir, "linked-dir")); err != nil {
		t.Fatal(err)
	}
	if err := writeRootFileAtomic(root, "linked-dir/new-file", []byte("new"), 0600); err == nil {
		t.Fatal("symlink parent accepted")
	}
}

func TestAtomicShortcutFileReplacesRegularFile(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("launcher", []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeRootFileAtomic(root, "launcher", []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := root.ReadFile("launcher")
	if err != nil || string(got) != "new" {
		t.Fatalf("replacement failed: %q %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files leaked: %v %v", entries, err)
	}
}

func TestQuitCancellationAbortsRestart(t *testing.T) {
	waited := false
	err := quitAndWait(AppNormal, func(AppConfig) error { return errors.New("user cancelled") }, func(AppConfig, time.Duration) bool { waited = true; return true })
	if err == nil || waited {
		t.Fatalf("cancelled quit must not proceed: error=%v waited=%v", err, waited)
	}
	if err := quitAndWait(AppNormal, func(AppConfig) error { return nil }, func(AppConfig, time.Duration) bool { return false }); err == nil {
		t.Fatal("timeout accepted as successful exit")
	}
	if err := quitAndWait(AppNormal, func(AppConfig) error { return nil }, func(AppConfig, time.Duration) bool { return true }); err != nil {
		t.Fatal(err)
	}
}

func TestMonitorLockConcurrentContenders(t *testing.T) {
	monitorStateDirOverride = t.TempDir()
	defer func() { monitorStateDirOverride = "" }()
	start := make(chan struct{})
	releases := make(chan func(), 32)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if release, ok := acquireMonitorLock(AppNormal); ok {
				releases <- release
			}
		}()
	}
	close(start)
	wg.Wait()
	close(releases)
	winners := len(releases)
	for release := range releases {
		release()
		release()
	}
	if winners != 1 {
		t.Fatalf("expected exactly one monitor, got %d", winners)
	}
}

func TestMonitorLockAcrossProcesses(t *testing.T) {
	monitorStateDirOverride = t.TempDir()
	defer func() { monitorStateDirOverride = "" }()
	release, ok := acquireMonitorLock(AppNormal)
	if !ok {
		t.Fatal("parent cannot acquire lock")
	}
	defer release()
	cmd := exec.Command(os.Args[0], "-test.run=^TestMonitorLockChild$")
	cmd.Env = append(os.Environ(), "AGY_HANS_LOCK_TEST_DIR="+monitorStateDirOverride)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("second process acquired held lock: %v\n%s", err, output)
	}
}

func TestMonitorLockChild(t *testing.T) {
	dir := os.Getenv("AGY_HANS_LOCK_TEST_DIR")
	if dir == "" {
		t.Skip("subprocess helper")
	}
	monitorStateDirOverride = dir
	if release, ok := acquireMonitorLock(AppNormal); ok {
		release()
		t.Fatal("lock held by parent must be unavailable")
	}
}

func TestMonitorIdentityRejectsPIDReuse(t *testing.T) {
	self, err := readSelfMonitorIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if !matchesMonitorIdentity(self) {
		t.Fatal("current identity rejected")
	}
	self.CreatedAt++
	if matchesMonitorIdentity(self) {
		t.Fatal("creation time mismatch accepted")
	}
}

func readSelfMonitorIdentity() (monitorIdentity, error) {
	p, err := os.Executable()
	if err != nil {
		return monitorIdentity{}, err
	}
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return monitorIdentity{}, err
	}
	created, err := proc.CreateTime()
	return monitorIdentity{PID: int32(os.Getpid()), CreatedAt: created, Executable: p}, err
}

func TestShellQuotePreservesLiteralPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX launcher is macOS-specific")
	}
	for _, s := range []string{"simple path", "dollar$HOME", "$(touch DO_NOT_EXECUTE)", "back`tick`", "a'b\"c", "line\nbreak", ""} {
		cmd := exec.Command("/bin/sh", "-c", "printf '%s' "+shellQuote(s))
		cmd.Dir = t.TempDir()
		out, err := cmd.Output()
		if err != nil || string(out) != s {
			t.Fatalf("quote changed %q into %q: %v", s, out, err)
		}
		entries, _ := os.ReadDir(cmd.Dir)
		if len(entries) != 0 {
			t.Fatal("path executed a command")
		}
	}
}

func TestMonitorLockRejectsSymlink(t *testing.T) {
	monitorStateDirOverride = t.TempDir()
	defer func() { monitorStateDirOverride = "" }()
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, monitorLockPath(AppNormal)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if release, ok := acquireMonitorLock(AppNormal); ok {
		release()
		t.Fatal("symlink lock accepted")
	}
	data, _ := os.ReadFile(victim)
	if string(data) != "original" {
		t.Fatal("lock changed unrelated file")
	}
}

func TestMonitorExecutableDoesNotTrustFilenameAlone(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if !isMonitorExecutable(self, self) {
		t.Fatal("own executable rejected")
	}
	fake := filepath.Join(t.TempDir(), "antigravity-hans-fake")
	if err := os.WriteFile(fake, []byte("not a Go monitor"), 0700); err != nil {
		t.Fatal(err)
	}
	if isMonitorExecutable(fake, self) {
		t.Fatal("process filename accepted without executable identity")
	}
}
