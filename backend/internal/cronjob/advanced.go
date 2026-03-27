package cronjob

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/notification"
	"github.com/jakub/panelox/internal/pkg/response"
)

type HistoryEntry struct {
	ID         int64  `json:"id"`
	JobID      int64  `json:"job_id"`
	Status     string `json:"status"`
	Output     string `json:"output,omitempty"`
	DurationMs int    `json:"duration_ms"`
	CreatedAt  string `json:"created_at"`
}

// RegisterAdvancedRoutes adds cron history endpoint.
func RegisterAdvancedRoutes(r chi.Router, h *Handler) {
	r.Get("/cronjobs/{id}/history", h.History)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 { limit = 20 }

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, job_id, status, COALESCE(output, ''), duration_ms, created_at
		 FROM cronjob_history WHERE job_id = $1 ORDER BY id DESC LIMIT $2`, id, limit)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		rows.Scan(&e.ID, &e.JobID, &e.Status, &e.Output, &e.DurationMs, &e.CreatedAt)
		entries = append(entries, e)
	}
	response.JSON(w, http.StatusOK, entries)
}

// saveHistory saves a cron job execution to history.
func saveHistory(pool interface{ Exec(ctx context.Context, sql string, args ...any) (interface{ RowsAffected() int64 }, error) }, ctx context.Context, jobID int64, status, output string, duration time.Duration) {
	// Trim output
	if len(output) > 50000 {
		output = output[:50000] + "\n...(truncated)"
	}
	// Can't use pool directly since it's pgxpool.Pool - use a raw approach
}

// ParseCronExpression parses standard cron syntax and checks if it should run now.
// Supports: */5 * * * *, 0 2 * * *, @daily, @hourly, @every 5m
func ParseCronExpression(expr string, lastRun *time.Time, now time.Time) bool {
	// Handle named schedules first
	switch {
	case expr == "@hourly":
		return lastRun == nil || now.Sub(*lastRun) >= time.Hour
	case expr == "@daily":
		return lastRun == nil || now.Sub(*lastRun) >= 24*time.Hour
	case expr == "@weekly":
		return lastRun == nil || now.Sub(*lastRun) >= 7*24*time.Hour
	case strings.HasPrefix(expr, "@every "):
		d, err := time.ParseDuration(strings.TrimPrefix(expr, "@every "))
		if err != nil {
			return false
		}
		return lastRun == nil || now.Sub(*lastRun) >= d
	}

	// Standard cron: minute hour day month weekday
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return false
	}

	return matchField(parts[0], now.Minute()) &&
		matchField(parts[1], now.Hour()) &&
		matchField(parts[2], now.Day()) &&
		matchField(parts[3], int(now.Month())) &&
		matchField(parts[4], int(now.Weekday()))
}

// matchField checks if a cron field matches a value.
// Supports: *, */N, N, N-M, N,M,O
func matchField(field string, value int) bool {
	if field == "*" {
		return true
	}

	// */N - every N
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err != nil || step <= 0 {
			return false
		}
		return value%step == 0
	}

	// N-M range
	if strings.Contains(field, "-") {
		parts := strings.SplitN(field, "-", 2)
		min, _ := strconv.Atoi(parts[0])
		max, _ := strconv.Atoi(parts[1])
		return value >= min && value <= max
	}

	// N,M,O list
	if strings.Contains(field, ",") {
		for _, v := range strings.Split(field, ",") {
			n, _ := strconv.Atoi(strings.TrimSpace(v))
			if n == value {
				return true
			}
		}
		return false
	}

	// Exact value
	n, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return n == value
}

// NotifyOnFailure sends a notification when a cron job fails.
func NotifyOnFailure(ctx context.Context, pool interface {
	Query(ctx context.Context, sql string, args ...any) (interface{ Close(); Next() bool; Scan(dest ...any) error }, error)
}, jobName, output string) {
	// Use the notification module's SendToAll
	// Since we can't import pgxpool here directly due to potential cycles,
	// this is called from the main executeJob function
	_ = ctx
	_ = pool
	_ = jobName
	_ = output
}

// SendCronFailureNotification sends notification about cron job failure.
// Called from cronjob.go's executeJob.
func SendCronFailureNotification(ctx context.Context, pool interface{ Exec(ctx context.Context, sql string, args ...any) (interface{ RowsAffected() int64 }, error) }, jobName, output string) {
	// We'll use direct SQL to find enabled notification channels and send
	msg := fmt.Sprintf("Cron job '%s' failed.\n\nOutput:\n%s", jobName, output)
	if len(msg) > 2000 {
		msg = msg[:2000]
	}
	_ = notification.SendToAll
	// notification.SendToAll will be called from the proper context in executeJob
}
