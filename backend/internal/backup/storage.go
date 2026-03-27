package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

// Storage account types
const (
	StorageLocal = "local"
	StorageS3    = "s3"
	StorageSFTP  = "sftp"
)

type BackupAccount struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	IsDefault bool            `json:"is_default"`
	CreatedAt string          `json:"created_at"`
}

type BackupSchedule struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"` // website, database, full
	AccountID      *int64 `json:"account_id,omitempty"`
	Schedule       string `json:"schedule"`
	RetentionDays  int    `json:"retention_days"`
	RetentionCount int    `json:"retention_count"`
	Enabled        bool   `json:"enabled"`
	LastRun        *string `json:"last_run,omitempty"`
	LastStatus     *string `json:"last_status,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// RegisterStorageRoutes adds backup account and schedule routes.
func RegisterStorageRoutes(r chi.Router, h *Handler) {
	r.Route("/backup-accounts", func(r chi.Router) {
		r.Get("/", h.ListAccounts)
		r.Post("/", h.CreateAccount)
		r.Delete("/{id}", h.DeleteAccount)
		r.Post("/{id}/test", h.TestAccount)
	})

	r.Route("/backup-schedules", func(r chi.Router) {
		r.Get("/", h.ListSchedules)
		r.Post("/", h.CreateSchedule)
		r.Delete("/{id}", h.DeleteSchedule)
		r.Post("/{id}/toggle", h.ToggleSchedule)
		r.Post("/{id}/run", h.RunScheduleNow)
	})
}

// --- Accounts ---

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, type, config, is_default, created_at FROM backup_accounts ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var accounts []BackupAccount
	for rows.Next() {
		var a BackupAccount
		rows.Scan(&a.ID, &a.Name, &a.Type, &a.Config, &a.IsDefault, &a.CreatedAt)
		accounts = append(accounts, a)
	}
	response.JSON(w, http.StatusOK, accounts)
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string          `json:"name"`
		Type   string          `json:"type"`
		Config json.RawMessage `json:"config"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" || body.Type == "" {
		response.BadRequest(w, "name and type are required")
		return
	}

	var a BackupAccount
	h.pool.QueryRow(r.Context(),
		`INSERT INTO backup_accounts (name, type, config) VALUES ($1, $2, $3)
		 RETURNING id, name, type, config, is_default, created_at`,
		body.Name, body.Type, body.Config,
	).Scan(&a.ID, &a.Name, &a.Type, &a.Config, &a.IsDefault, &a.CreatedAt)

	response.JSON(w, http.StatusCreated, a)
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM backup_accounts WHERE id = $1 AND is_default = false", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "account deleted"})
}

func (h *Handler) TestAccount(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var accType string
	var config json.RawMessage
	h.pool.QueryRow(r.Context(), "SELECT type, config FROM backup_accounts WHERE id = $1", id).Scan(&accType, &config)

	err := testStorage(accType, config)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Schedules ---

func (h *Handler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, type, account_id, schedule, retention_days, retention_count, enabled, last_run, last_status, created_at FROM backup_schedules ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var schedules []BackupSchedule
	for rows.Next() {
		var s BackupSchedule
		rows.Scan(&s.ID, &s.Name, &s.Type, &s.AccountID, &s.Schedule, &s.RetentionDays, &s.RetentionCount, &s.Enabled, &s.LastRun, &s.LastStatus, &s.CreatedAt)
		schedules = append(schedules, s)
	}
	response.JSON(w, http.StatusOK, schedules)
}

func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string `json:"name"`
		Type           string `json:"type"`
		AccountID      *int64 `json:"account_id"`
		Schedule       string `json:"schedule"`
		RetentionDays  int    `json:"retention_days"`
		RetentionCount int    `json:"retention_count"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" || body.Type == "" {
		response.BadRequest(w, "name and type are required")
		return
	}
	if body.Schedule == "" { body.Schedule = "@daily" }
	if body.RetentionDays == 0 { body.RetentionDays = 30 }
	if body.RetentionCount == 0 { body.RetentionCount = 10 }

	var s BackupSchedule
	h.pool.QueryRow(r.Context(),
		`INSERT INTO backup_schedules (name, type, account_id, schedule, retention_days, retention_count)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, type, account_id, schedule, retention_days, retention_count, enabled, last_run, last_status, created_at`,
		body.Name, body.Type, body.AccountID, body.Schedule, body.RetentionDays, body.RetentionCount,
	).Scan(&s.ID, &s.Name, &s.Type, &s.AccountID, &s.Schedule, &s.RetentionDays, &s.RetentionCount, &s.Enabled, &s.LastRun, &s.LastStatus, &s.CreatedAt)

	response.JSON(w, http.StatusCreated, s)
}

func (h *Handler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM backup_schedules WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "schedule deleted"})
}

func (h *Handler) ToggleSchedule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "UPDATE backup_schedules SET enabled = NOT enabled WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}

func (h *Handler) RunScheduleNow(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	go runScheduledBackup(h.pool, id)
	response.JSON(w, http.StatusAccepted, map[string]string{"message": "backup started"})
}

// --- Storage upload functions ---

func uploadToStorage(accType string, config json.RawMessage, localPath, remoteName string) error {
	var cfg map[string]string
	json.Unmarshal(config, &cfg)

	switch accType {
	case StorageS3:
		return uploadToS3(cfg, localPath, remoteName)
	case StorageSFTP:
		return uploadToSFTP(cfg, localPath, remoteName)
	case StorageLocal:
		// Already local, just copy
		destDir := cfg["path"]
		if destDir == "" { destDir = defaultBackupDir }
		dest := filepath.Join(destDir, remoteName)
		out, err := exec.Command("cp", localPath, dest).CombinedOutput()
		if err != nil { return fmt.Errorf("cp: %s", string(out)) }
		return nil
	default:
		return fmt.Errorf("unsupported storage type: %s", accType)
	}
}

func uploadToS3(cfg map[string]string, localPath, remoteName string) error {
	// Use AWS CLI (most reliable, handles auth, regions, etc.)
	bucket := cfg["bucket"]
	region := cfg["region"]
	endpoint := cfg["endpoint"] // For MinIO/B2 custom endpoint
	prefix := cfg["prefix"]

	remotePath := fmt.Sprintf("s3://%s/%s%s", bucket, prefix, remoteName)

	args := []string{"s3", "cp", localPath, remotePath}
	if region != "" {
		args = append(args, "--region", region)
	}

	cmd := exec.Command("aws", args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", cfg["access_key"]),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", cfg["secret_key"]),
	)
	if endpoint != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ENDPOINT_URL=%s", endpoint))
		args = append(args, "--endpoint-url", endpoint)
		cmd = exec.Command("aws", args...)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", cfg["access_key"]),
			fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", cfg["secret_key"]),
		)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aws s3 cp: %s", string(out))
	}
	return nil
}

func uploadToSFTP(cfg map[string]string, localPath, remoteName string) error {
	host := cfg["host"]
	port := cfg["port"]
	user := cfg["user"]
	password := cfg["password"]
	keyFile := cfg["key_file"]
	remotePath := cfg["path"]

	if port == "" { port = "22" }
	if remotePath == "" { remotePath = "/backups" }

	dest := fmt.Sprintf("%s@%s:%s/%s", user, host, remotePath, remoteName)

	var args []string
	if keyFile != "" {
		args = []string{"-i", keyFile, "-P", port, "-o", "StrictHostKeyChecking=no", localPath, dest}
	} else {
		// Use sshpass for password auth
		args = []string{"-p", password, "scp", "-P", port, "-o", "StrictHostKeyChecking=no", localPath, dest}
		out, err := exec.Command("sshpass", args...).CombinedOutput()
		if err != nil { return fmt.Errorf("sshpass scp: %s", string(out)) }
		return nil
	}

	out, err := exec.Command("scp", args...).CombinedOutput()
	if err != nil { return fmt.Errorf("scp: %s", string(out)) }
	return nil
}

func testStorage(accType string, config json.RawMessage) error {
	var cfg map[string]string
	json.Unmarshal(config, &cfg)

	switch accType {
	case StorageLocal:
		path := cfg["path"]
		if path == "" { path = defaultBackupDir }
		return os.MkdirAll(path, 0755)
	case StorageS3:
		// Test by listing bucket
		bucket := cfg["bucket"]
		cmd := exec.Command("aws", "s3", "ls", fmt.Sprintf("s3://%s", bucket), "--max-items", "1")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", cfg["access_key"]),
			fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", cfg["secret_key"]),
		)
		if cfg["endpoint"] != "" {
			cmd.Args = append(cmd.Args, "--endpoint-url", cfg["endpoint"])
		}
		if cfg["region"] != "" {
			cmd.Args = append(cmd.Args, "--region", cfg["region"])
		}
		out, err := cmd.CombinedOutput()
		if err != nil { return fmt.Errorf("S3 test failed: %s", string(out)) }
		return nil
	case StorageSFTP:
		host := cfg["host"]
		port := cfg["port"]
		user := cfg["user"]
		if port == "" { port = "22" }
		out, err := exec.Command("ssh", "-p", port, "-o", "StrictHostKeyChecking=no",
			"-o", "ConnectTimeout=5", fmt.Sprintf("%s@%s", user, host), "echo ok").CombinedOutput()
		if err != nil { return fmt.Errorf("SSH test failed: %s", string(out)) }
		return nil
	}
	return fmt.Errorf("unknown type: %s", accType)
}

// --- Scheduled backup runner ---

func runScheduledBackup(pool *pgxpool.Pool, scheduleID int64) {
	ctx := context.Background()

	var sched BackupSchedule
	var accountID *int64
	pool.QueryRow(ctx,
		"SELECT id, name, type, account_id, retention_days, retention_count FROM backup_schedules WHERE id = $1", scheduleID,
	).Scan(&sched.ID, &sched.Name, &sched.Type, &accountID, &sched.RetentionDays, &sched.RetentionCount)

	pool.Exec(ctx, "UPDATE backup_schedules SET last_run = NOW(), last_status = 'running' WHERE id = $1", scheduleID)

	timestamp := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("%s-%s", sched.Name, timestamp)
	localPath := filepath.Join(defaultBackupDir, name+".tar.gz")

	// Create backup locally
	b := &Backup{
		Name:     name,
		Type:     sched.Type,
		FilePath: localPath,
		Status:   "running",
	}
	runBackup(pool, b)

	// Upload to remote if account specified
	if accountID != nil {
		var accType string
		var config json.RawMessage
		pool.QueryRow(ctx, "SELECT type, config FROM backup_accounts WHERE id = $1", *accountID).Scan(&accType, &config)

		if accType != StorageLocal {
			if err := uploadToStorage(accType, config, localPath, name+".tar.gz"); err != nil {
				log.Printf("Backup upload failed: %v", err)
				pool.Exec(ctx, "UPDATE backup_schedules SET last_status = 'upload_failed' WHERE id = $1", scheduleID)
				return
			}
		}
	}

	pool.Exec(ctx, "UPDATE backup_schedules SET last_status = 'success' WHERE id = $1", scheduleID)

	// Apply retention policy
	applyRetention(pool, ctx, sched.Name, sched.RetentionDays, sched.RetentionCount)
}

func applyRetention(pool *pgxpool.Pool, ctx context.Context, namePrefix string, maxDays, maxCount int) {
	// Delete backups older than retention_days
	if maxDays > 0 {
		pool.Exec(ctx,
			"DELETE FROM backups WHERE name LIKE $1 AND created_at < NOW() - ($2 || ' days')::interval",
			namePrefix+"%", fmt.Sprintf("%d", maxDays))
	}

	// Keep only retention_count most recent
	if maxCount > 0 {
		pool.Exec(ctx,
			`DELETE FROM backups WHERE name LIKE $1 AND id NOT IN (
				SELECT id FROM backups WHERE name LIKE $1 ORDER BY created_at DESC LIMIT $2
			)`, namePrefix+"%", maxCount)
	}

	// Clean up orphan files
	entries, _ := os.ReadDir(defaultBackupDir)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), namePrefix) { continue }
		info, _ := e.Info()
		if info != nil && time.Since(info.ModTime()).Hours() > float64(maxDays*24) {
			os.Remove(filepath.Join(defaultBackupDir, e.Name()))
		}
	}
}

// BackupScheduleRunner checks and runs due scheduled backups every hour.
type ScheduleRunner struct {
	pool *pgxpool.Pool
	stop chan struct{}
}

func NewScheduleRunner(pool *pgxpool.Pool) *ScheduleRunner {
	return &ScheduleRunner{pool: pool, stop: make(chan struct{})}
}

func (sr *ScheduleRunner) Start() {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sr.checkDue()
			case <-sr.stop:
				return
			}
		}
	}()
	log.Println("Backup schedule runner started")
}

func (sr *ScheduleRunner) Stop() { close(sr.stop) }

func (sr *ScheduleRunner) checkDue() {
	ctx := context.Background()
	rows, err := sr.pool.Query(ctx,
		"SELECT id, schedule, last_run FROM backup_schedules WHERE enabled = true")
	if err != nil { return }
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var id int64
		var schedule string
		var lastRun *time.Time
		rows.Scan(&id, &schedule, &lastRun)

		var interval time.Duration
		switch {
		case schedule == "@hourly": interval = time.Hour
		case schedule == "@daily": interval = 24 * time.Hour
		case schedule == "@weekly": interval = 7 * 24 * time.Hour
		case strings.HasPrefix(schedule, "@every "):
			d, err := time.ParseDuration(strings.TrimPrefix(schedule, "@every "))
			if err != nil { continue }
			interval = d
		default: interval = 24 * time.Hour
		}

		if lastRun == nil || now.Sub(*lastRun) >= interval {
			go runScheduledBackup(sr.pool, id)
		}
	}
}
