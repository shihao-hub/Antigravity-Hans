package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const maxCDPJSONSize = 1 << 20
const maxCDPMessageSize = 8 << 20

var cdpHTTPClient = &http.Client{
	Timeout:       500 * time.Millisecond,
	Transport:     &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 500 * time.Millisecond}).DialContext, IdleConnTimeout: 30 * time.Second},
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
}

func readCDPJSON(port int, endpoint string, dst interface{}) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("无效的 CDP 端口")
	}
	resp, err := cdpHTTPClient.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, endpoint))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CDP HTTP 状态码: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCDPJSONSize+1))
	if err != nil {
		return err
	}
	if len(body) > maxCDPJSONSize {
		return fmt.Errorf("CDP 响应超出体积上限")
	}
	return json.Unmarshal(body, dst)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateCDPWebSocket(raw string, expectedPort int) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port != expectedPort || port <= 0 || port > 65535 || u.Scheme != "ws" ||
		(u.Hostname() != "127.0.0.1" && !strings.EqualFold(u.Hostname(), "localhost")) || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(!strings.HasPrefix(u.Path, "/devtools/page/") && !strings.HasPrefix(u.Path, "/devtools/browser/")) {
		return "", fmt.Errorf("拒绝调试端口之外的 WebSocket 地址")
	}
	// Avoid resolving localhost through DNS or inheriting an HTTP proxy.
	if strings.EqualFold(u.Hostname(), "localhost") {
		u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	}
	return u.String(), nil
}

func dialCDP(raw string, port int) (*websocket.Conn, *http.Response, error) {
	validated, err := validateCDPWebSocket(raw, port)
	if err != nil {
		return nil, nil, err
	}
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second, Proxy: nil, NetDialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext}
	conn, resp, err := dialer.Dial(validated, nil)
	if err == nil {
		conn.SetReadLimit(maxCDPMessageSize)
	}
	return conn, resp, err
}
