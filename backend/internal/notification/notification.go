package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Channel struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"` // telegram, discord, email, webhook
	Config    json.RawMessage `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt string          `json:"created_at"`
}

type CreateChannelRequest struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/notifications", func(r chi.Router) {
		r.Get("/channels", h.ListChannels)
		r.Post("/channels", h.CreateChannel)
		r.Delete("/channels/{id}", h.DeleteChannel)
		r.Post("/channels/{id}/toggle", h.ToggleChannel)
		r.Post("/channels/{id}/test", h.TestChannel)
	})
}

func (h *Handler) ListChannels(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, type, config, enabled, created_at FROM notification_channels ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var channels []Channel
	for rows.Next() {
		var ch Channel
		rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled, &ch.CreatedAt)
		channels = append(channels, ch)
	}
	response.JSON(w, http.StatusOK, channels)
}

func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" {
		response.BadRequest(w, "name and type are required")
		return
	}

	var ch Channel
	h.pool.QueryRow(r.Context(),
		`INSERT INTO notification_channels (name, type, config)
		 VALUES ($1, $2, $3) RETURNING id, name, type, config, enabled, created_at`,
		req.Name, req.Type, req.Config,
	).Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled, &ch.CreatedAt)

	response.JSON(w, http.StatusCreated, ch)
}

func (h *Handler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM notification_channels WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "channel deleted"})
}

func (h *Handler) ToggleChannel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "UPDATE notification_channels SET enabled = NOT enabled WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}

func (h *Handler) TestChannel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var ch Channel
	h.pool.QueryRow(r.Context(),
		"SELECT id, name, type, config FROM notification_channels WHERE id = $1", id,
	).Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config)

	err := Send(ch.Type, ch.Config, "Panelox Test", "This is a test notification from Panelox.")
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "test notification sent"})
}

// Send dispatches a notification to the appropriate channel.
func Send(channelType string, config json.RawMessage, title, message string) error {
	var cfg map[string]string
	json.Unmarshal(config, &cfg)

	switch channelType {
	case "telegram":
		return sendTelegram(cfg["bot_token"], cfg["chat_id"], title, message)
	case "discord":
		return sendDiscord(cfg["webhook_url"], title, message)
	case "email":
		return sendEmail(cfg["smtp_host"], cfg["smtp_port"], cfg["username"], cfg["password"], cfg["to"], title, message)
	case "webhook":
		return sendWebhook(cfg["url"], title, message)
	default:
		return fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

func sendTelegram(botToken, chatID, title, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	body, _ := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       fmt.Sprintf("*%s*\n%s", title, message),
		"parse_mode": "Markdown",
	})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API returned %d", resp.StatusCode)
	}
	return nil
}

func sendDiscord(webhookURL, title, message string) error {
	body, _ := json.Marshal(map[string]any{
		"embeds": []map[string]any{
			{"title": title, "description": message, "color": 3447003},
		},
	})
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord returned %d", resp.StatusCode)
	}
	return nil
}

func sendEmail(host, port, username, password, to, title, message string) error {
	auth := smtp.PlainAuth("", username, password, host)
	msg := fmt.Sprintf("Subject: %s\r\nContent-Type: text/plain\r\n\r\n%s", title, message)
	addr := fmt.Sprintf("%s:%s", host, port)
	return smtp.SendMail(addr, auth, username, []string{to}, []byte(msg))
}

func sendWebhook(url, title, message string) error {
	body, _ := json.Marshal(map[string]string{"title": title, "message": message})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SendToAll sends a notification to all enabled channels.
func SendToAll(ctx context.Context, pool *pgxpool.Pool, title, message string) {
	rows, err := pool.Query(ctx,
		"SELECT type, config FROM notification_channels WHERE enabled = true")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var chType string
		var config json.RawMessage
		rows.Scan(&chType, &config)
		if err := Send(chType, config, title, message); err != nil {
			log.Printf("Notification send failed (%s): %v", chType, err)
		}
	}
}
