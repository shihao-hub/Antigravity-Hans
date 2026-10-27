package main

import (
	"os"
	"testing"
)

// Opt-in, read-only verification. Never launches, quits, or injects into an app.
func TestRunningNormalAppEndpoint(t *testing.T) {
	if os.Getenv("AGY_HANS_TEST_RUNNING_APP") != "1" {
		t.Skip("set AGY_HANS_TEST_RUNNING_APP=1 with the normal app open")
	}
	port, ok := findCDPPort(AppNormal)
	if !ok {
		t.Fatal("cannot identify an owned normal-app CDP listener")
	}
	if _, err := getBrowserWSURL(port); err != nil {
		t.Fatal(err)
	}
	targets := CheckPort(port)
	matched := false
	for _, target := range targets {
		if isTargetPage(target.URL, AppNormal.Name) {
			matched = true
		}
	}
	if !matched {
		t.Fatal("normal app has no matching application page")
	}
	t.Logf("verified owned ordinary-app CDP endpoint on port %d", port)
}
