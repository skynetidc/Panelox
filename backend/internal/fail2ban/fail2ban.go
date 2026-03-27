package fail2ban

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Rule struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Port     string `json:"port"`
	MaxRetry int    `json:"max_retry"`
	BanTime  int    `json:"ban_time"`
	FindTime int    `json:"find_time"`
	Enabled  bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

type BannedIP struct {
	ID        int64  `json:"id"`
	RuleID    int64  `json:"rule_id"`
	IPAddress string `json:"ip_address"`
	Reason    string `json:"reason"`
	BannedAt  string `json:"banned_at"`
	ExpiresAt string `json:"expires_at"`
}

type CreateRuleRequest struct {
	Name     string `json:"name"`
	Port     string `json:"port"`
	MaxRetry int    `json:"max_retry"`
	BanTime  int    `json:"ban_time"`
	FindTime int    `json:"find_time"`
}

// In-memory tracker for failed attempts
type attemptTracker struct {
	mu       sync.Mutex
	attempts map[string][]time.Time // ip -> timestamps
}

var tracker = &attemptTracker{attempts: make(map[string][]time.Time)}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/fail2ban", func(r chi.Router) {
		r.Get("/rules", h.ListRules)
		r.Post("/rules", h.CreateRule)
		r.Delete("/rules/{id}", h.DeleteRule)
		r.Post("/rules/{id}/toggle", h.ToggleRule)
		r.Get("/banned", h.ListBanned)
		r.Post("/unban", h.Unban)
	})
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, port, max_retry, ban_time, find_time, enabled, created_at FROM fail2ban_rules ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rule Rule
		rows.Scan(&rule.ID, &rule.Name, &rule.Port, &rule.MaxRetry, &rule.BanTime, &rule.FindTime, &rule.Enabled, &rule.CreatedAt)
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
	if req.Name == "" || req.Port == "" {
		response.BadRequest(w, "name and port are required")
		return
	}
	if req.MaxRetry == 0 { req.MaxRetry = 5 }
	if req.BanTime == 0 { req.BanTime = 3600 }
	if req.FindTime == 0 { req.FindTime = 600 }

	var rule Rule
	h.pool.QueryRow(r.Context(),
		`INSERT INTO fail2ban_rules (name, port, max_retry, ban_time, find_time)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, port, max_retry, ban_time, find_time, enabled, created_at`,
		req.Name, req.Port, req.MaxRetry, req.BanTime, req.FindTime,
	).Scan(&rule.ID, &rule.Name, &rule.Port, &rule.MaxRetry, &rule.BanTime, &rule.FindTime, &rule.Enabled, &rule.CreatedAt)

	response.JSON(w, http.StatusCreated, rule)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM fail2ban_rules WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "rule deleted"})
}

func (h *Handler) ToggleRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "UPDATE fail2ban_rules SET enabled = NOT enabled WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}

func (h *Handler) ListBanned(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, rule_id, ip_address, reason, banned_at, expires_at
		 FROM fail2ban_banned WHERE expires_at > NOW() ORDER BY banned_at DESC`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var banned []BannedIP
	for rows.Next() {
		var b BannedIP
		rows.Scan(&b.ID, &b.RuleID, &b.IPAddress, &b.Reason, &b.BannedAt, &b.ExpiresAt)
		banned = append(banned, b)
	}
	response.JSON(w, http.StatusOK, banned)
}

func (h *Handler) Unban(w http.ResponseWriter, r *http.Request) {
	var body struct{ IP string `json:"ip"` }
	json.NewDecoder(r.Body).Decode(&body)
	if body.IP == "" {
		response.BadRequest(w, "ip is required")
		return
	}

	// Remove from DB
	h.pool.Exec(r.Context(), "DELETE FROM fail2ban_banned WHERE ip_address = $1", body.IP)

	// Remove iptables rule
	exec.Command("iptables", "-D", "INPUT", "-s", body.IP, "-j", "DROP").Run()

	response.JSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("unbanned %s", body.IP)})
}

// RecordFailedAttempt records a failed login attempt and bans IP if threshold exceeded.
func RecordFailedAttempt(ctx context.Context, pool *pgxpool.Pool, ip string) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	now := time.Now()
	tracker.attempts[ip] = append(tracker.attempts[ip], now)

	// Check all enabled rules
	rows, err := pool.Query(ctx,
		"SELECT id, port, max_retry, ban_time, find_time FROM fail2ban_rules WHERE enabled = true")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ruleID int64
		var port string
		var maxRetry, banTime, findTime int
		rows.Scan(&ruleID, &port, &maxRetry, &banTime, &findTime)

		// Count attempts within find_time window
		cutoff := now.Add(-time.Duration(findTime) * time.Second)
		count := 0
		for _, t := range tracker.attempts[ip] {
			if t.After(cutoff) {
				count++
			}
		}

		if count >= maxRetry {
			banIP(ctx, pool, ruleID, ip, banTime)
			// Clean up tracked attempts
			delete(tracker.attempts, ip)
			break
		}
	}
}

func banIP(ctx context.Context, pool *pgxpool.Pool, ruleID int64, ip string, banTime int) {
	expiresAt := time.Now().Add(time.Duration(banTime) * time.Second)

	pool.Exec(ctx,
		`INSERT INTO fail2ban_banned (rule_id, ip_address, reason, expires_at)
		 VALUES ($1, $2, 'too many failed attempts', $3)`,
		ruleID, ip, expiresAt)

	// Block via iptables
	exec.Command("iptables", "-A", "INPUT", "-s", ip, "-j", "DROP").Run()

	log.Printf("Fail2Ban: banned IP %s for %d seconds", ip, banTime)

	// Schedule unban
	go func() {
		time.Sleep(time.Duration(banTime) * time.Second)
		exec.Command("iptables", "-D", "INPUT", "-s", ip, "-j", "DROP").Run()
		pool.Exec(context.Background(), "DELETE FROM fail2ban_banned WHERE ip_address = $1 AND expires_at <= NOW()", ip)
		log.Printf("Fail2Ban: unbanned IP %s", ip)
	}()
}
