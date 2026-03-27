package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"golang.org/x/crypto/ssh"
)

const LocalSockPath = "/var/run/panelox-agent.sock"

// NewLocalProxy creates a reverse proxy to the local agent via Unix socket.
func NewLocalProxy() *httputil.ReverseProxy {
	target, _ := url.Parse("http://panelox-agent")
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", LocalSockPath, 10*time.Second)
		},
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Local proxy error: %v", err)
		http.Error(w, `{"error":{"code":"proxy_error","message":"local agent unavailable"}}`, http.StatusBadGateway)
	}

	return proxy
}

// NewRemoteProxy creates a reverse proxy to a remote agent via SSH tunnel.
func NewRemoteProxy(sshClient *ssh.Client, agentPort int) *httputil.ReverseProxy {
	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", agentPort))
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return sshClient.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", agentPort))
		},
		MaxIdleConns:       5,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Remote proxy error: %v", err)
		http.Error(w, `{"error":{"code":"proxy_error","message":"remote agent unavailable"}}`, http.StatusBadGateway)
	}

	return proxy
}

// ProxyRequest forwards an HTTP request to a remote agent via SSH.
func ProxyRequest(sshClient *ssh.Client, agentPort int, method, path string, body io.Reader) (*http.Response, error) {
	conn, err := sshClient.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", agentPort))
	if err != nil {
		return nil, fmt.Errorf("SSH dial: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return conn, nil
			},
		},
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", agentPort, path), body)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
