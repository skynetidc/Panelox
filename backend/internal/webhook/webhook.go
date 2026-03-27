package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

var ValidEvents = []string{
	"website.created", "website.deleted", "website.started", "website.stopped",
	"ssl.issued", "ssl.renewed", "ssl.expiring",
	"backup.completed", "backup.failed",
	"database.created", "database.deleted",
	"user.created", "user.deleted",
	"deploy.success", "deploy.failed",
	"alert.firing", "alert.resolved",
}

type Webhook struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Secret     string   `json:"secret,omitempty"`
	Events     []string `json:"events"`
	Enabled    bool     `json:"enabled"`
	LastSent   *string  `json:"last_sent,omitempty"`
	LastStatus *int     `json:"last_status,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/webhooks", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/events", h.ListEvents)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/toggle", h.Toggle)
		r.Post("/{id}/test", h.Test)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, url, secret, events, enabled, last_sent, last_status, created_at FROM webhooks ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()
	var hooks []Webhook
	for rows.Next() {
		var wh Webhook
		rows.Scan(&wh.ID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events, &wh.Enabled, &wh.LastSent, &wh.LastStatus, &wh.CreatedAt)
		hooks = append(hooks, wh)
	}
	response.JSON(w, http.StatusOK, hooks)
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, ValidEvents)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string   `json:"name"`
		URL    string   `json:"url"`
		Secret string   `json:"secret"`
		Events []string `json:"events"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" || body.URL == "" {
		response.BadRequest(w, "name and url are required")
		return
	}

	var wh Webhook
	h.pool.QueryRow(r.Context(),
		`INSERT INTO webhooks (name, url, secret, events) VALUES ($1, $2, $3, $4)
		 RETURNING id, name, url, secret, events, enabled, created_at`,
		body.Name, body.URL, body.Secret, body.Events,
	).Scan(&wh.ID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events, &wh.Enabled, &wh.CreatedAt)
	response.JSON(w, http.StatusCreated, wh)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM webhooks WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "webhook deleted"})
}

func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "UPDATE webhooks SET enabled = NOT enabled WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}

func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var wh Webhook
	h.pool.QueryRow(r.Context(), "SELECT url, secret FROM webhooks WHERE id = $1", id).Scan(&wh.URL, &wh.Secret)
	err := sendWebhook(wh.URL, wh.Secret, "test", map[string]any{"message": "test webhook from Panelox"})
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Dispatch sends an event to all matching enabled webhooks.
func Dispatch(ctx context.Context, pool *pgxpool.Pool, event string, data any) {
	rows, err := pool.Query(ctx,
		"SELECT id, url, secret, events FROM webhooks WHERE enabled = true")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var url, secret string
		var events []string
		rows.Scan(&id, &url, &secret, &events)

		matches := false
		for _, e := range events {
			if e == event || e == "*" {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}

		go func(whID int64, whURL, whSecret string) {
			err := sendWebhook(whURL, whSecret, event, data)
			status := 200
			if err != nil {
				status = 0
				log.Printf("Webhook %d failed: %v", whID, err)
			}
			pool.Exec(context.Background(),
				"UPDATE webhooks SET last_sent = NOW(), last_status = $1 WHERE id = $2", status, whID)
		}(id, url, secret)
	}
}

func sendWebhook(url, secret, event string, data any) error {
	payload := map[string]any{"event": event, "data": data, "timestamp": time.Now().Unix()}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Panelox-Event", event)

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Panelox-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
