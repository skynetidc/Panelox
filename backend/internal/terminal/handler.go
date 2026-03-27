package terminal

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
	"nhooyr.io/websocket"
)

type Handler struct {
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/terminal/ws", h.HandleWebSocket)
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}
	if c.Role != "admin" {
		response.Forbidden(w, "terminal requires admin role")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := r.Context()

	// Find shell
	shell := "/bin/bash"
	if _, err := exec.LookPath(shell); err != nil {
		shell = "/bin/sh"
	}

	// Start shell with PTY
	cmd := exec.CommandContext(ctx, shell, "--login")
	cmd.Env = []string{
		"TERM=xterm-256color",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/root",
		"USER=root",
		"SHELL=" + shell,
		"LANG=en_US.UTF-8",
	}
	cmd.Dir = "/root"

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("PTY start error: %v", err)
		conn.Write(ctx, websocket.MessageBinary, []byte("Failed to start terminal: "+err.Error()+"\r\n"))
		return
	}
	defer ptmx.Close()

	// Set initial size
	pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	var wg sync.WaitGroup

	// PTY output -> WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if writeErr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket input -> PTY (handle resize messages)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			// Check for resize JSON message
			if len(data) > 0 && data[0] == '{' {
				var msg struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" {
					pty.Setsize(ptmx, &pty.Winsize{Rows: msg.Rows, Cols: msg.Cols})
					continue
				}
			}
			ptmx.Write(data)
		}
	}()

	wg.Wait()
	cmd.Process.Kill()
	cmd.Wait()
}

// Ensure os is used
var _ = os.Getenv
