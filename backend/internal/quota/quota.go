package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

// --- Models ---

type Quota struct {
	ID              int64 `json:"id"`
	UserID          int64 `json:"user_id"`
	MaxWebsites     int   `json:"max_websites"`
	MaxDatabases    int   `json:"max_databases"`
	MaxFTPUsers     int   `json:"max_ftp_users"`
	MaxMailDomains  int   `json:"max_mail_domains"`
	MaxDiskMB       int64 `json:"max_disk_mb"`
	MaxBandwidthMB  int64 `json:"max_bandwidth_mb"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type QuotaUsage struct {
	Quota
	UsedWebsites    int   `json:"used_websites"`
	UsedDatabases   int   `json:"used_databases"`
	UsedFTPUsers    int   `json:"used_ftp_users"`
	UsedMailDomains int   `json:"used_mail_domains"`
	UsedDiskMB      int64 `json:"used_disk_mb"`
}

type Package struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	MaxWebsites     int     `json:"max_websites"`
	MaxDatabases    int     `json:"max_databases"`
	MaxFTPUsers     int     `json:"max_ftp_users"`
	MaxMailDomains  int     `json:"max_mail_domains"`
	MaxDiskMB       int64   `json:"max_disk_mb"`
	MaxBandwidthMB  int64   `json:"max_bandwidth_mb"`
	MaxUsers        int     `json:"max_users"`
	PriceMonthly    float64 `json:"price_monthly"`
	CreatedAt       string  `json:"created_at"`
}

type UserBillingInfo struct {
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	Suspended bool   `json:"suspended"`
	Package   string `json:"package,omitempty"`
	Quota     *QuotaUsage `json:"quota,omitempty"`
}

// --- Handler ---

type Handler struct {
	pool      *pgxpool.Pool
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(pool *pgxpool.Pool, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{pool: pool, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/quotas", func(r chi.Router) {
		r.Get("/user/{userId}", h.GetUserQuota)
		r.Put("/user/{userId}", h.SetUserQuota)
		r.Get("/my", h.GetMyQuota)
	})

	r.Route("/packages", func(r chi.Router) {
		r.Get("/", h.ListPackages)
		r.Post("/", h.CreatePackage)
		r.Put("/{id}", h.UpdatePackage)
		r.Delete("/{id}", h.DeletePackage)
		r.Post("/{id}/assign/{userId}", h.AssignPackage)
	})

	r.Post("/users/{id}/suspend", h.SuspendUser)
	r.Post("/users/{id}/unsuspend", h.UnsuspendUser)
	r.Get("/users/billing", h.BillingDashboard)
}

// --- Quotas ---

func (h *Handler) GetUserQuota(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	usage, err := h.getUserQuotaUsage(r.Context(), userID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, usage)
}

func (h *Handler) GetMyQuota(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil { response.Unauthorized(w, "not authenticated"); return }
	usage, err := h.getUserQuotaUsage(r.Context(), c.UserID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, usage)
}

func (h *Handler) SetUserQuota(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)

	var q Quota
	json.NewDecoder(r.Body).Decode(&q)

	_, err := h.pool.Exec(r.Context(),
		`INSERT INTO user_quotas (user_id, max_websites, max_databases, max_ftp_users, max_mail_domains, max_disk_mb, max_bandwidth_mb)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id) DO UPDATE SET
			max_websites = $2, max_databases = $3, max_ftp_users = $4, max_mail_domains = $5,
			max_disk_mb = $6, max_bandwidth_mb = $7, updated_at = NOW()`,
		userID, q.MaxWebsites, q.MaxDatabases, q.MaxFTPUsers, q.MaxMailDomains, q.MaxDiskMB, q.MaxBandwidthMB)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "quota updated"})
}

func (h *Handler) getUserQuotaUsage(ctx context.Context, userID int64) (*QuotaUsage, error) {
	usage := &QuotaUsage{Quota: Quota{UserID: userID}}

	// Get quotas (or defaults if not set)
	err := h.pool.QueryRow(ctx,
		`SELECT COALESCE(max_websites, 0), COALESCE(max_databases, 0), COALESCE(max_ftp_users, 0),
			COALESCE(max_mail_domains, 0), COALESCE(max_disk_mb, 0), COALESCE(max_bandwidth_mb, 0)
		 FROM user_quotas WHERE user_id = $1`, userID,
	).Scan(&usage.MaxWebsites, &usage.MaxDatabases, &usage.MaxFTPUsers, &usage.MaxMailDomains, &usage.MaxDiskMB, &usage.MaxBandwidthMB)
	if err == pgx.ErrNoRows {
		// No quota set = unlimited (0)
	} else if err != nil {
		return nil, err
	}

	// Count used resources
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM websites WHERE user_id = $1", userID).Scan(&usage.UsedWebsites)
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM db_servers WHERE user_id = $1", userID).Scan(&usage.UsedDatabases)
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM ftp_users fu JOIN websites w ON fu.website_id = w.id WHERE w.user_id = $1", userID).Scan(&usage.UsedFTPUsers)
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM mail_domains WHERE user_id = $1", userID).Scan(&usage.UsedMailDomains)

	// Disk usage
	usage.UsedDiskMB = getUserDiskMB(h.pool, ctx, userID)

	return usage, nil
}

// CheckQuota returns error if the user has exceeded their quota for the given resource.
func CheckQuota(ctx context.Context, pool *pgxpool.Pool, userID int64, resource string) error {
	var max, used int
	var err error

	switch resource {
	case "websites":
		pool.QueryRow(ctx, "SELECT COALESCE(max_websites, 0) FROM user_quotas WHERE user_id = $1", userID).Scan(&max)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM websites WHERE user_id = $1", userID).Scan(&used)
	case "databases":
		pool.QueryRow(ctx, "SELECT COALESCE(max_databases, 0) FROM user_quotas WHERE user_id = $1", userID).Scan(&max)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM db_servers WHERE user_id = $1", userID).Scan(&used)
	case "ftp_users":
		pool.QueryRow(ctx, "SELECT COALESCE(max_ftp_users, 0) FROM user_quotas WHERE user_id = $1", userID).Scan(&max)
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM ftp_users fu JOIN websites w ON fu.website_id = w.id WHERE w.user_id = $1", userID).Scan(&used)
	case "mail_domains":
		pool.QueryRow(ctx, "SELECT COALESCE(max_mail_domains, 0) FROM user_quotas WHERE user_id = $1", userID).Scan(&max)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM mail_domains WHERE user_id = $1", userID).Scan(&used)
	default:
		return nil
	}

	if err != nil {
		return nil // If quota check fails, allow the action
	}

	if max > 0 && used >= max {
		return fmt.Errorf("quota exceeded: %s (%d/%d)", resource, used, max)
	}
	return nil
}

// --- Reseller Packages ---

func (h *Handler) ListPackages(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, max_websites, max_databases, max_ftp_users, max_mail_domains, max_disk_mb, max_bandwidth_mb, max_users, price_monthly, created_at
		 FROM reseller_packages ORDER BY price_monthly`)
	if err != nil { response.InternalError(w, err.Error()); return }
	defer rows.Close()

	var packages []Package
	for rows.Next() {
		var p Package
		rows.Scan(&p.ID, &p.Name, &p.MaxWebsites, &p.MaxDatabases, &p.MaxFTPUsers, &p.MaxMailDomains, &p.MaxDiskMB, &p.MaxBandwidthMB, &p.MaxUsers, &p.PriceMonthly, &p.CreatedAt)
		packages = append(packages, p)
	}
	response.JSON(w, http.StatusOK, packages)
}

func (h *Handler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var p Package
	json.NewDecoder(r.Body).Decode(&p)
	if p.Name == "" { response.BadRequest(w, "name is required"); return }

	h.pool.QueryRow(r.Context(),
		`INSERT INTO reseller_packages (name, max_websites, max_databases, max_ftp_users, max_mail_domains, max_disk_mb, max_bandwidth_mb, max_users, price_monthly)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at`,
		p.Name, p.MaxWebsites, p.MaxDatabases, p.MaxFTPUsers, p.MaxMailDomains, p.MaxDiskMB, p.MaxBandwidthMB, p.MaxUsers, p.PriceMonthly,
	).Scan(&p.ID, &p.CreatedAt)

	response.JSON(w, http.StatusCreated, p)
}

func (h *Handler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var p Package
	json.NewDecoder(r.Body).Decode(&p)

	h.pool.Exec(r.Context(),
		`UPDATE reseller_packages SET name=$1, max_websites=$2, max_databases=$3, max_ftp_users=$4, max_mail_domains=$5,
			max_disk_mb=$6, max_bandwidth_mb=$7, max_users=$8, price_monthly=$9 WHERE id=$10`,
		p.Name, p.MaxWebsites, p.MaxDatabases, p.MaxFTPUsers, p.MaxMailDomains, p.MaxDiskMB, p.MaxBandwidthMB, p.MaxUsers, p.PriceMonthly, id)

	response.JSON(w, http.StatusOK, map[string]string{"message": "package updated"})
}

func (h *Handler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM reseller_packages WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "package deleted"})
}

func (h *Handler) AssignPackage(w http.ResponseWriter, r *http.Request) {
	pkgID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID, _ := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)

	// Get package limits
	var p Package
	err := h.pool.QueryRow(r.Context(),
		"SELECT max_websites, max_databases, max_ftp_users, max_mail_domains, max_disk_mb, max_bandwidth_mb FROM reseller_packages WHERE id = $1", pkgID,
	).Scan(&p.MaxWebsites, &p.MaxDatabases, &p.MaxFTPUsers, &p.MaxMailDomains, &p.MaxDiskMB, &p.MaxBandwidthMB)
	if err != nil { response.NotFound(w, "package not found"); return }

	// Set user package + quotas
	h.pool.Exec(r.Context(), "UPDATE users SET package_id = $1 WHERE id = $2", pkgID, userID)
	h.pool.Exec(r.Context(),
		`INSERT INTO user_quotas (user_id, max_websites, max_databases, max_ftp_users, max_mail_domains, max_disk_mb, max_bandwidth_mb)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id) DO UPDATE SET
			max_websites=$2, max_databases=$3, max_ftp_users=$4, max_mail_domains=$5, max_disk_mb=$6, max_bandwidth_mb=$7, updated_at=NOW()`,
		userID, p.MaxWebsites, p.MaxDatabases, p.MaxFTPUsers, p.MaxMailDomains, p.MaxDiskMB, p.MaxBandwidthMB)

	response.JSON(w, http.StatusOK, map[string]string{"message": "package assigned"})
}

// --- User Suspension ---

func (h *Handler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct{ Reason string `json:"reason"` }
	json.NewDecoder(r.Body).Decode(&body)

	// Suspend user
	h.pool.Exec(r.Context(),
		"UPDATE users SET suspended = true, suspended_at = NOW(), suspend_reason = $1, status = 'suspended' WHERE id = $2",
		body.Reason, userID)

	// Stop all user's websites
	rows, _ := h.pool.Query(r.Context(), "SELECT id FROM websites WHERE user_id = $1 AND status = 'active'", userID)
	if rows != nil {
		for rows.Next() {
			var webID int64
			rows.Scan(&webID)
			h.pool.Exec(r.Context(), "UPDATE websites SET status = 'stopped' WHERE id = $1", webID)
		}
		rows.Close()
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("user %d suspended", userID)})
}

func (h *Handler) UnsuspendUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	h.pool.Exec(r.Context(),
		"UPDATE users SET suspended = false, suspended_at = NULL, suspend_reason = NULL, status = 'active' WHERE id = $1", userID)

	// Restart websites
	h.pool.Exec(r.Context(),
		"UPDATE websites SET status = 'active' WHERE user_id = $1 AND status = 'stopped'", userID)

	response.JSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("user %d unsuspended", userID)})
}

// --- Billing Dashboard ---

func (h *Handler) BillingDashboard(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT u.id, u.name, u.email, u.role, u.status, COALESCE(u.suspended, false),
			COALESCE(rp.name, '')
		 FROM users u
		 LEFT JOIN reseller_packages rp ON u.package_id = rp.id
		 ORDER BY u.id`)
	if err != nil { response.InternalError(w, err.Error()); return }
	defer rows.Close()

	var users []UserBillingInfo
	for rows.Next() {
		var u UserBillingInfo
		rows.Scan(&u.UserID, &u.Name, &u.Email, &u.Role, &u.Status, &u.Suspended, &u.Package)

		usage, _ := h.getUserQuotaUsage(r.Context(), u.UserID)
		u.Quota = usage
		users = append(users, u)
	}

	response.JSON(w, http.StatusOK, users)
}

func getUserDiskMB(pool *pgxpool.Pool, ctx context.Context, userID int64) int64 {
	rows, _ := pool.Query(ctx, "SELECT alias FROM websites WHERE user_id = $1", userID)
	if rows == nil { return 0 }
	defer rows.Close()

	var totalBytes int64
	for rows.Next() {
		var alias string
		rows.Scan(&alias)
		out, err := exec.Command("du", "-sb", fmt.Sprintf("/www/sites/%s", alias)).Output()
		if err == nil {
			parts := strings.Fields(string(out))
			if len(parts) >= 1 { b, _ := strconv.ParseInt(parts[0], 10, 64); totalBytes += b }
		}
	}
	return totalBytes / (1024 * 1024)
}

// Compile checks
var _ = time.Now
