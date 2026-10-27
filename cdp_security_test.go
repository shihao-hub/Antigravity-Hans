package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"
)

func serverPort(s *httptest.Server) int { return s.Listener.Addr().(*net.TCPAddr).Port }

func TestNormalAppURLRequiresLoopbackHost(t *testing.T) {
	for _, raw := range []string{"https://localhost.example.invalid/", "https://example.invalid/?next=http://localhost:3000", "file:///tmp/localhost.html", "https://127.0.0.1@example.invalid/"} {
		if isTargetPage(raw, AppNormal.Name) {
			t.Errorf("remote or non-app page accepted: %s", raw)
		}
	}
}

func TestCDPWebSocketBoundary(t *testing.T) {
	for _, raw := range []string{"ws://example.invalid:58321/devtools/page/a", "ws://localhost.example.invalid:58321/devtools/page/a", "ws://127.0.0.1:58322/devtools/page/a", "ws://user@127.0.0.1:58321/devtools/page/a", "ws://127.0.0.1:58321/unrelated", "ws://127.0.0.1:58321/devtools/page/a?redirect=1", "wss://127.0.0.1:58321/devtools/page/a"} {
		if _, err := validateCDPWebSocket(raw, 58321); err == nil {
			t.Errorf("unsafe URL accepted: %s", raw)
		}
	}
	if got, err := validateCDPWebSocket("ws://localhost:58321/devtools/page/a", 58321); err != nil || !strings.Contains(got, "127.0.0.1:58321") {
		t.Fatalf("localhost not pinned to loopback: %s %v", got, err)
	}
	if _, err := validateCDPWebSocket("ws://127.0.0.2:58321/devtools/page/a", 58321); err == nil {
		t.Fatal("different loopback host accepted")
	}
}

func TestCDPJSONRejectsRedirectsAndOversizedResponses(t *testing.T) {
	var followed atomic.Int32
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); _, _ = fmt.Fprint(w, `[]`) }))
	defer foreign.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, foreign.URL, http.StatusFound) }))
	defer redirect.Close()
	var dst interface{}
	if err := readCDPJSON(serverPort(redirect), "/json/list", &dst); err == nil || followed.Load() != 0 {
		t.Fatal("CDP followed a redirect")
	}
	large := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, strings.Repeat(" ", maxCDPJSONSize+1))
	}))
	defer large.Close()
	if err := readCDPJSON(serverPort(large), "/json/list", &dst); err == nil {
		t.Fatal("oversized response accepted")
	}
}

func TestCheckPortRejectsForeignWebsocket(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]CDPTarget{{ID: "a", Type: "page", URL: "https://127.0.0.1/app", WebSocketDebuggerURL: "ws://example.invalid:58321/devtools/page/a"}})
	}))
	defer s.Close()
	if got := CheckPort(serverPort(s)); len(got) != 0 {
		t.Fatal("foreign WebSocket target accepted")
	}
}

func TestFindPortRequiresOwnerAndMatchingApp(t *testing.T) {
	var s *httptest.Server
	var pageURL atomic.Value
	pageURL.Store("https://127.0.0.1:3000/app")
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]CDPTarget{{ID: "a", Type: "page", URL: pageURL.Load().(string), WebSocketDebuggerURL: fmt.Sprintf("ws://127.0.0.1:%d/devtools/page/a", serverPort(s))}})
	}))
	defer s.Close()
	port := serverPort(s)
	cfg := AppConfig{Name: AppNormal.Name, Port: port}
	if _, ok := findCDPPortWithOwner(cfg, func(int) bool { return false }); ok {
		t.Fatal("unowned endpoint reused")
	}
	if got, ok := findCDPPortWithOwner(cfg, func(p int) bool { return p == port }); !ok || got != port {
		t.Fatal("owned normal app not reused")
	}
	pageURL.Store("file:///Applications/Antigravity IDE.app/out/workbench.html")
	if _, ok := findCDPPortWithOwner(cfg, func(p int) bool { return p == port }); ok {
		t.Fatal("IDE endpoint accepted for normal app")
	}
}

func TestNormalAppCanReuseOwnedRandomPort(t *testing.T) {
	var s *httptest.Server
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]CDPTarget{{ID: "a", Type: "page", URL: "http://127.0.0.1/app", WebSocketDebuggerURL: fmt.Sprintf("ws://127.0.0.1:%d/devtools/page/a", serverPort(s))}})
	}))
	defer s.Close()
	port := serverPort(s)
	if got, ok := findCDPPortWithOwner(AppNormal, func(p int) bool { return p == port }, port); !ok || got != port {
		t.Fatal("normal app native random port not reused")
	}
}

func TestBrowserWSURLRejectsOffHostAddress(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"webSocketDebuggerUrl": "ws://example.invalid/devtools/browser/foreign"})
	}))
	defer s.Close()
	if _, err := getBrowserWSURL(serverPort(s)); err == nil {
		t.Fatal("foreign browser address accepted")
	}
}

func TestNormalAppReusesOwnedBrowserWithNoOpenWindows(t *testing.T) {
	var s *httptest.Server
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json/list" {
			_, _ = fmt.Fprint(w, "[]")
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"webSocketDebuggerUrl": fmt.Sprintf("ws://127.0.0.1:%d/devtools/browser/live", serverPort(s))})
	}))
	defer s.Close()
	port := serverPort(s)
	if got, ok := findCDPPortWithOwner(AppNormal, func(p int) bool { return p == port }, port); !ok || got != port {
		t.Fatal("background normal app would be unnecessarily restarted")
	}
}

func TestEnsureTargetInjectedProtocolAndVerification(t *testing.T) {
	var installed atomic.Bool
	var registrations atomic.Int32
	var legacy atomic.Bool
	var fingerprint atomic.Value
	fingerprint.Store("")
	upgrader := websocket.Upgrader{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var cmd struct {
				ID     int64                  `json:"id"`
				Method string                 `json:"method"`
				Params map[string]interface{} `json:"params"`
			}
			if conn.ReadJSON(&cmd) != nil {
				return
			}
			result := map[string]interface{}{}
			switch cmd.Method {
			case "Page.addScriptToEvaluateOnNewDocument":
				registrations.Add(1)
			case "Runtime.evaluate":
				expression, _ := cmd.Params["expression"].(string)
				if cmd.Params["returnByValue"] == true {
					matched := installed.Load()
					if strings.Contains(expression, "Cleanup") {
						matched = matched && legacy.Load()
					} else if strings.Contains(expression, "PatchFingerprint") {
						matched = matched && fingerprint.Load().(string) != "" && strings.Contains(expression, fmt.Sprintf("%q", fingerprint.Load().(string)))
					}
					result["result"] = map[string]interface{}{"value": matched}
				} else {
					installed.Store(true)
					prefix, _, _ := strings.Cut(expression, ";\n")
					_, raw, _ := strings.Cut(prefix, " = ")
					var value string
					_ = json.Unmarshal([]byte(raw), &value)
					fingerprint.Store(value)
				}
			}
			if conn.WriteJSON(map[string]interface{}{"id": cmd.ID, "result": result}) != nil {
				return
			}
		}
	}))
	defer s.Close()
	target := CDPTarget{ID: "a", DebugPort: serverPort(s), WebSocketDebuggerURL: fmt.Sprintf("ws://127.0.0.1:%d/devtools/page/a", serverPort(s))}
	if ok, _, err := EnsureTargetInjected(target, string(embeddedOverlayJS)); !ok || err != nil {
		t.Fatalf("injection not verified: %v %v", ok, err)
	}
	if ok, _, err := EnsureTargetInjected(target, string(embeddedOverlayJS)); !ok || err != nil {
		t.Fatal("installed document not recognized")
	}
	if registrations.Load() != 1 {
		t.Fatalf("duplicate registration: %d", registrations.Load())
	}
	if ok, _, err := EnsureTargetInjected(target, string(embeddedOverlayJS)+"\n// dictionary update"); !ok || err != nil {
		t.Fatalf("new dictionary not reinstalled: %v", err)
	}
	if registrations.Load() != 2 {
		t.Fatal("new fingerprint did not replace the old dictionary")
	}
	legacy.Store(true)
	fingerprint.Store("")
	if ok, permanent, err := EnsureTargetInjected(target, string(embeddedOverlayJS)); ok || !permanent || err == nil || !strings.Contains(err.Error(), "重新加载") {
		t.Fatalf("legacy observer must require reload: %v %v %v", ok, permanent, err)
	}
}
