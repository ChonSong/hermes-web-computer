package xpra

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"nhooyr.io/websocket"
	"time"
)

// ProxyHandler relays the frontend's HTTP requests to the Xpra HTML5 server.
type ProxyHandler struct {
	mgr *Manager
}

// NewProxyHandler creates a proxy handler for a given Xpra manager.
func NewProxyHandler(mgr *Manager) *ProxyHandler {
	return &ProxyHandler{mgr: mgr}
}

// ServeHTTP proxies all requests to the Xpra HTML5 server.
func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(p.mgr.HTTPURL())
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid Xpra URL: %v", err), 500)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = target
			req.Host = target.Host
		},
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	proxy.ServeHTTP(w, r)
}

// HTML5Index serves the Xpra HTML5 client entry point.
func (p *ProxyHandler) HTML5Index(w http.ResponseWriter, r *http.Request) {
	if !p.mgr.IsRunning() {
		http.Error(w, "Xpra server not running", 503)
		return
	}

	// Redirect to the Xpra HTML5 client
	display := p.mgr.Display()
	targetURL := fmt.Sprintf("%s/index.html?session=%s", p.mgr.HTTPURL(), display)

	http.Redirect(w, r, targetURL, http.StatusFound)
}

// WaitForServer waits for the Xpra HTML5 server to be ready.
func (p *ProxyHandler) WaitForServer(timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(p.mgr.HTTPURL())
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("Xpra HTML5 server not ready within %v", timeout)
}

// WsProxyHandler handles WebSocket proxying for the Xpra HTML5 client.
// The xpra HTML5 client connects via WebSocket to the server for screen updates.
type WsProxyHandler struct {
	mgr *Manager
}

// NewWsProxyHandler creates a WebSocket proxy handler for Xpra.
func NewWsProxyHandler(mgr *Manager) *WsProxyHandler {
	return &WsProxyHandler{mgr: mgr}
}

// ServeWs proxies WebSocket connections from the browser to the Xpra server.
// Xpra uses WebSockets for the HTML5 client communication.
func (p *WsProxyHandler) ServeWs(w http.ResponseWriter, r *http.Request) {
	if !p.mgr.IsRunning() {
		http.Error(w, "Xpra server not running", 503)
		return
	}

	// Build the target WebSocket URL for Xpra
	wsURL := fmt.Sprintf("ws://localhost:%d/", p.mgr.HTTPPort())

	// Dial the Xpra WebSocket server
	dialCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	targetConn, _, err := websocket.Dial(dialCtx, wsURL, nil)
	if err != nil {
		log.Printf("[xpra-ws] failed to connect to Xpra WebSocket: %v", err)
		http.Error(w, fmt.Sprintf("failed to connect to Xpra: %v", err), 502)
		return
	}
	defer targetConn.Close(websocket.StatusNormalClosure, "proxy closing")

	// Accept the browser's WebSocket connection
	browserConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("[xpra-ws] failed to accept browser connection: %v", err)
		return
	}
	defer browserConn.Close(websocket.StatusNormalClosure, "proxy closing")

	// Bidirectional copy between browser and Xpra
	errCh := make(chan error, 2)

	go func() {
		for {
			msgType, msg, err := browserConn.Read(r.Context())
			if err != nil {
				errCh <- err
				return
			}
			if err := targetConn.Write(r.Context(), msgType, msg); err != nil {
				errCh <- err
				return
			}
		}
	}()

	go func() {
		for {
			msgType, msg, err := targetConn.Read(r.Context())
			if err != nil {
				errCh <- err
				return
			}
			if err := browserConn.Write(r.Context(), msgType, msg); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Wait for either direction to fail
	<-errCh
}

// copyLoop copies data between two readers/writers.
func copyLoop(dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			return err
		}
	}
}
