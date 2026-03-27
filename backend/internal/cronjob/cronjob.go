package cronjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

type CronJob struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Schedule   string     `json:"schedule"`
	Command    string     `json:"command"`
	Enabled    bool       `json:"enabled"`
	LastRun    *time.Time `json:"last_run,omitempty"`
	LastStatus string     `json:"last_status,omitempty"`
	LastOutput string     `json:"last_output,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
}

type Handler struct {
	pool      *pgxpool.Pool
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(pool *pgxpool.Pool, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{pool: pool, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/cronjobs", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/run", h.RunNow)
		r.Post("/{id}/toggle", h.Toggle)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, user_id, name, type, schedule, command, enabled, last_run, last_status, last_output, created_at, updated_at
		 FROM cronjobs ORDER BY id DESC`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var jobs []CronJob
	for rows.Next() {
		var j CronJob
		rows.Scan(&j.ID, &j.UserID, &j.Name, &j.Type, &j.Schedule, &j.Command, &j.Enabled, &j.LastRun, &j.LastStatus, &j.LastOutput, &j.CreatedAt, &j.UpdatedAt)
		jobs = append(jobs, j)
	}
	response.JSON(w, http.StatusOK, jobs)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Schedule == "" || req.Command == "" {
		response.BadRequest(w, "name, schedule and command are required")
		return
	}
	if req.Type == "" {
		req.Type = "shell"
	}

	var j CronJob
	err := h.pool.QueryRow(r.Context(),
		`INSERT INTO cronjobs (user_id, name, type, schedule, command) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, name, type, schedule, command, enabled, created_at, updated_at`,
		c.UserID, req.Name, req.Type, req.Schedule, req.Command,
	).Scan(&j.ID, &j.UserID, &j.Name, &j.Type, &j.Schedule, &j.Command, &j.Enabled, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, j)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body map[string]any
	json.NewDecoder(r.Body).Decode(&body)

	sets := []string{"updated_at = NOW()"}
	args := []any{}
	idx := 1
	for _, key := range []string{"name", "schedule", "command", "type"} {
		if v, ok := body[key]; ok {
			sets = append(sets, fmt.Sprintf("%s = $%d", key, idx))
			args = append(args, v)
			idx++
		}
	}
	args = append(args, id)
	h.pool.Exec(r.Context(), fmt.Sprintf("UPDATE cronjobs SET %s WHERE id = $%d", strings.Join(sets, ", "), idx), args...)
	response.JSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM cronjobs WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "UPDATE cronjobs SET enabled = NOT enabled, updated_at = NOW() WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}

func (h *Handler) RunNow(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var j CronJob
	err := h.pool.QueryRow(r.Context(),
		"SELECT id, command, type FROM cronjobs WHERE id = $1", id,
	).Scan(&j.ID, &j.Command, &j.Type)
	if err == pgx.ErrNoRows {
		response.NotFound(w, "job not found")
		return
	}

	go executeJob(h.pool, &j)
	response.JSON(w, http.StatusAccepted, map[string]string{"message": "execution started"})
}

// Scheduler runs enabled cron jobs based on their schedule.
type Scheduler struct {
	pool *pgxpool.Pool
	stop chan struct{}
	mu   sync.Mutex
}

func NewScheduler(pool *pgxpool.Pool) *Scheduler {
	return &Scheduler{pool: pool, stop: make(chan struct{})}
}

func (s *Scheduler) Start() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.checkAndRun()
			case <-s.stop:
				return
			}
		}
	}()
	log.Println("Cron scheduler started")
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) checkAndRun() {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx,
		`SELECT id, command, type, schedule, last_run FROM cronjobs WHERE enabled = true`)
	if err != nil {
		return
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var j CronJob
		rows.Scan(&j.ID, &j.Command, &j.Type, &j.Schedule, &j.LastRun)

		if shouldRun(j.Schedule, j.LastRun, now) {
			go executeJob(s.pool, &j)
		}
	}
}

func executeJob(pool *pgxpool.Pool, j *CronJob) {
	ctx := context.Background()
	start := time.Now()

	var cmd *exec.Cmd
	switch j.Type {
	case "curl":
		cmd = exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", j.Command)
	default: // shell
		cmd = exec.Command("/bin/sh", "-c", j.Command)
	}

	out, err := cmd.CombinedOutput()
	status := "success"
	output := string(out)
	if err != nil {
		status = "failed"
		output = fmt.Sprintf("%s\nError: %v", output, err)
	}

	// Trim output to 10KB
	if len(output) > 10240 {
		output = output[:10240] + "\n... (truncated)"
	}

	pool.Exec(ctx,
		`UPDATE cronjobs SET last_run = $1, last_status = $2, last_output = $3, updated_at = NOW() WHERE id = $4`,
		start, status, output, j.ID)

	log.Printf("Cron job %d (%s) completed in %s: %s", j.ID, j.Name, time.Since(start).Round(time.Millisecond), status)
}

// shouldRun checks if a cron job should run based on simple interval parsing.
// Supports: @every 5m, @every 1h, @hourly, @daily, @weekly
func shouldRun(schedule string, lastRun *time.Time, now time.Time) bool {
	if lastRun == nil {
		return true
	}

	var interval time.Duration
	switch {
	case schedule == "@hourly":
		interval = time.Hour
	case schedule == "@daily":
		interval = 24 * time.Hour
	case schedule == "@weekly":
		interval = 7 * 24 * time.Hour
	case strings.HasPrefix(schedule, "@every "):
		d, err := time.ParseDuration(strings.TrimPrefix(schedule, "@every "))
		if err != nil {
			return false
		}
		interval = d
	default:
		// Standard cron: run every minute check (simplified)
		interval = time.Minute
	}

	return now.Sub(*lastRun) >= interval
}

var ErrNotFound = errors.New("not found")
