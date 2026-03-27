package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/notification"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Rule struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Metric        string     `json:"metric"` // cpu, memory, disk, load
	Operator      string     `json:"operator"` // >, <, >=, <=
	Threshold     float64    `json:"threshold"`
	DurationSec   int        `json:"duration_sec"`
	ChannelID     *int64     `json:"channel_id,omitempty"`
	Enabled       bool       `json:"enabled"`
	LastTriggered *string    `json:"last_triggered,omitempty"`
	Status        string     `json:"status"` // ok, firing
	CreatedAt     string     `json:"created_at"`
}

type CreateRuleRequest struct {
	Name        string  `json:"name"`
	Metric      string  `json:"metric"`
	Operator    string  `json:"operator"`
	Threshold   float64 `json:"threshold"`
	DurationSec int     `json:"duration_sec"`
	ChannelID   *int64  `json:"channel_id,omitempty"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/alerts", func(r chi.Router) {
		r.Get("/rules", h.ListRules)
		r.Post("/rules", h.CreateRule)
		r.Delete("/rules/{id}", h.DeleteRule)
		r.Post("/rules/{id}/toggle", h.ToggleRule)
	})
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, metric, operator, threshold, duration_sec, channel_id, enabled, last_triggered, status, created_at FROM alert_rules ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rule Rule
		rows.Scan(&rule.ID, &rule.Name, &rule.Metric, &rule.Operator, &rule.Threshold, &rule.DurationSec, &rule.ChannelID, &rule.Enabled, &rule.LastTriggered, &rule.Status, &rule.CreatedAt)
		rules = append(rules, rule)
	}
	response.JSON(w, http.StatusOK, rules)
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Metric == "" {
		response.BadRequest(w, "name and metric are required")
		return
	}
	if req.Operator == "" { req.Operator = ">" }
	if req.DurationSec == 0 { req.DurationSec = 300 }

	var rule Rule
	h.pool.QueryRow(r.Context(),
		`INSERT INTO alert_rules (name, metric, operator, threshold, duration_sec, channel_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, metric, operator, threshold, duration_sec, channel_id, enabled, last_triggered, status, created_at`,
		req.Name, req.Metric, req.Operator, req.Threshold, req.DurationSec, req.ChannelID,
	).Scan(&rule.ID, &rule.Name, &rule.Metric, &rule.Operator, &rule.Threshold, &rule.DurationSec, &rule.ChannelID, &rule.Enabled, &rule.LastTriggered, &rule.Status, &rule.CreatedAt)

	response.JSON(w, http.StatusCreated, rule)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM alert_rules WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "rule deleted"})
}

func (h *Handler) ToggleRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "UPDATE alert_rules SET enabled = NOT enabled WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}

// Checker runs in the background and evaluates alert rules every minute.
type Checker struct {
	pool *pgxpool.Pool
	stop chan struct{}
}

func NewChecker(pool *pgxpool.Pool) *Checker {
	return &Checker{pool: pool, stop: make(chan struct{})}
}

func (c *Checker) Start() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.evaluate()
			case <-c.stop:
				return
			}
		}
	}()
	log.Println("Alert checker started")
}

func (c *Checker) Stop() { close(c.stop) }

func (c *Checker) evaluate() {
	ctx := context.Background()
	metrics := collectCurrentMetrics()

	rows, err := c.pool.Query(ctx,
		"SELECT id, name, metric, operator, threshold, channel_id, status FROM alert_rules WHERE enabled = true")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name, metric, operator, status string
		var threshold float64
		var channelID *int64
		rows.Scan(&id, &name, &metric, &operator, &threshold, &channelID, &status)

		value, ok := metrics[metric]
		if !ok {
			continue
		}

		firing := evaluate(value, operator, threshold)

		if firing && status != "firing" {
			// Transition to firing → send alert
			c.pool.Exec(ctx,
				"UPDATE alert_rules SET status = 'firing', last_triggered = NOW() WHERE id = $1", id)

			msg := fmt.Sprintf("Alert: %s\n%s is %.1f%% (threshold: %s %.1f%%)",
				name, metric, value, operator, threshold)

			if channelID != nil {
				var chType string
				var config json.RawMessage
				c.pool.QueryRow(ctx,
					"SELECT type, config FROM notification_channels WHERE id = $1", *channelID,
				).Scan(&chType, &config)
				notification.Send(chType, config, "Panelox Alert: "+name, msg)
			} else {
				notification.SendToAll(ctx, c.pool, "Panelox Alert: "+name, msg)
			}

			log.Printf("ALERT FIRING: %s - %s=%.1f%% %s %.1f%%", name, metric, value, operator, threshold)

		} else if !firing && status == "firing" {
			// Resolved
			c.pool.Exec(ctx, "UPDATE alert_rules SET status = 'ok' WHERE id = $1", id)
			log.Printf("ALERT RESOLVED: %s", name)
		}
	}
}

func evaluate(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	default:
		return value > threshold
	}
}

func collectCurrentMetrics() map[string]float64 {
	m := map[string]float64{}

	// CPU from load average
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			load, _ := strconv.ParseFloat(parts[0], 64)
			cores := 1.0
			if data2, err := os.ReadFile("/proc/cpuinfo"); err == nil {
				cores = float64(strings.Count(string(data2), "processor"))
				if cores < 1 { cores = 1 }
			}
			m["cpu"] = (load / cores) * 100
		}
	}

	// Memory
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, available int64
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				switch parts[0] {
				case "MemTotal:":
					total = val
				case "MemAvailable:":
					available = val
				}
			}
		}
		if total > 0 {
			m["memory"] = float64(total-available) / float64(total) * 100
		}
	}

	// Disk
	if data, err := os.ReadFile("/proc/mounts"); err == nil {
		_ = data
	}

	return m
}
