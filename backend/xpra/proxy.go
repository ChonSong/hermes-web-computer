package xpra

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(p.mgr.HTTPURL())
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