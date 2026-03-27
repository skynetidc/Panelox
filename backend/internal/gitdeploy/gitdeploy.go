package gitdeploy

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

type GitDeploy struct {
	ID            int64      `json:"id"`
	WebsiteID     int64      `json:"website_id"`
	RepoURL       string     `json:"repo_url"`
	Branch        string     `json:"branch"`
	AutoDeploy    bool       `json:"auto_deploy"`
	WebhookSecret string     `json:"webhook_secret"`
	BuildCommand  string     `json:"build_command"`
	DeployPath    string     `json:"deploy_path"`
	LastCommit    string     `json:"last_commit,omitempty"`
	LastDeploy    *time.Time `json:"last_deploy,omitempty"`
	LastStatus    string     `json:"last_status,omitempty"`
	LastLog       string     `json:"last_log,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type DeployHistoryEntry struct {
	ID         int64  `json:"id"`
	DeployID   int64  `json:"deploy_id"`
	CommitHash string `json:"commit_hash"`
	CommitMsg  string `json:"commit_msg"`
	Status     string `json:"status"`
	Log        string `json:"log,omitempty"`
	DurationMs int    `json:"duration_ms"`
	CreatedAt  string `json:"created_at"`
}

type CreateRequest struct {
	WebsiteID    int64  `json:"website_id"`
	RepoURL      string `json:"repo_url"`
	Branch       string `json:"branch"`
	BuildCommand string `json:"build_command"`
	AutoDeploy   bool   `json:"auto_deploy"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/git-deploy", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/deploy", h.TriggerDeploy)
		r.Post("/{id}/rollback", h.Rollback)
		r.Get("/{id}/history", h.History)
	})
}

// RegisterWebhookRoute registers the public webhook endpoint (no auth).
func RegisterWebhookRoute(r chi.Router, h *Handler) {
	r.Post("/webhook/git/{id}", h.WebhookReceiver)
}

func generateSecret() string {
	b := make([]byte, 20)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, website_id, repo_url, branch, auto_deploy, webhook_secret, build_command, deploy_path,
			last_commit, last_deploy, last_status, '', created_at
		 FROM git_deploys ORDER BY id DESC`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var deploys []GitDeploy
	for rows.Next() {
		var d GitDeploy
		rows.Scan(&d.ID, &d.WebsiteID, &d.RepoURL, &d.Branch, &d.AutoDeploy, &d.WebhookSecret,
			&d.BuildCommand, &d.DeployPath, &d.LastCommit, &d.LastDeploy, &d.LastStatus, &d.LastLog, &d.CreatedAt)
		deploys = append(deploys, d)
	}
	response.JSON(w, http.StatusOK, deploys)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var d GitDeploy
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, website_id, repo_url, branch, auto_deploy, webhook_secret, build_command, deploy_path,
			last_commit, last_deploy, last_status, last_log, created_at
		 FROM git_deploys WHERE id = $1`, id,
	).Scan(&d.ID, &d.WebsiteID, &d.RepoURL, &d.Branch, &d.AutoDeploy, &d.WebhookSecret,
		&d.BuildCommand, &d.DeployPath, &d.LastCommit, &d.LastDeploy, &d.LastStatus, &d.LastLog, &d.CreatedAt)
	if err == pgx.ErrNoRows {
		response.NotFound(w, "deploy config not found")
		return
	}
	response.JSON(w, http.StatusOK, d)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.WebsiteID == 0 || req.RepoURL == "" {
		response.BadRequest(w, "website_id and repo_url are required")
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}

	// Get website deploy path
	var alias string
	h.pool.QueryRow(r.Context(), "SELECT alias FROM websites WHERE id = $1", req.WebsiteID).Scan(&alias)
	deployPath := fmt.Sprintf("/www/sites/%s/index", alias)

	secret := generateSecret()
	var d GitDeploy
	h.pool.QueryRow(r.Context(),
		`INSERT INTO git_deploys (website_id, repo_url, branch, auto_deploy, webhook_secret, build_command, deploy_path)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, website_id, repo_url, branch, auto_deploy, webhook_secret, build_command, deploy_path, created_at`,
		req.WebsiteID, req.RepoURL, req.Branch, req.AutoDeploy, secret, req.BuildCommand, deployPath,
	).Scan(&d.ID, &d.WebsiteID, &d.RepoURL, &d.Branch, &d.AutoDeploy, &d.WebhookSecret, &d.BuildCommand, &d.DeployPath, &d.CreatedAt)

	response.JSON(w, http.StatusCreated, d)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM git_deploys WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "deploy config deleted"})
}

func (h *Handler) TriggerDeploy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	go executeDeploy(h.pool, id, "manual trigger", "")
	response.JSON(w, http.StatusAccepted, map[string]string{"message": "deploy started"})
}

func (h *Handler) Rollback(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct{ CommitHash string `json:"commit_hash"` }
	json.NewDecoder(r.Body).Decode(&body)

	if body.CommitHash == "" {
		// Get previous deploy
		h.pool.QueryRow(r.Context(),
			"SELECT commit_hash FROM deploy_history WHERE deploy_id = $1 AND status = 'success' ORDER BY id DESC OFFSET 1 LIMIT 1", id,
		).Scan(&body.CommitHash)
	}

	if body.CommitHash == "" {
		response.BadRequest(w, "no previous deploy to rollback to")
		return
	}

	go executeDeploy(h.pool, id, "rollback", body.CommitHash)
	response.JSON(w, http.StatusAccepted, map[string]string{"message": "rollback started", "commit": body.CommitHash})
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, deploy_id, commit_hash, commit_msg, status, '', duration_ms, created_at
		 FROM deploy_history WHERE deploy_id = $1 ORDER BY id DESC LIMIT 20`, id)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var entries []DeployHistoryEntry
	for rows.Next() {
		var e DeployHistoryEntry
		rows.Scan(&e.ID, &e.DeployID, &e.CommitHash, &e.CommitMsg, &e.Status, &e.Log, &e.DurationMs, &e.CreatedAt)
		entries = append(entries, e)
	}
	response.JSON(w, http.StatusOK, entries)
}

// WebhookReceiver handles incoming GitHub/GitLab webhooks (public, no JWT).
func (h *Handler) WebhookReceiver(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var deploy GitDeploy
	err := h.pool.QueryRow(r.Context(),
		"SELECT id, webhook_secret, branch, auto_deploy FROM git_deploys WHERE id = $1", id,
	).Scan(&deploy.ID, &deploy.WebhookSecret, &deploy.Branch, &deploy.AutoDeploy)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	if !deploy.AutoDeploy {
		w.WriteHeader(200)
		w.Write([]byte(`{"message":"auto-deploy disabled"}`))
		return
	}

	// Verify signature (GitHub: X-Hub-Signature-256)
	body, _ := io.ReadAll(r.Body)
	sig := r.Header.Get("X-Hub-Signature-256")
	if sig != "" && deploy.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(deploy.WebhookSecret))
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			http.Error(w, "invalid signature", 403)
			return
		}
	}

	// Parse push event
	var payload struct {
		Ref    string `json:"ref"`
		After  string `json:"after"`
		HeadCommit struct {
			Message string `json:"message"`
		} `json:"head_commit"`
	}
	json.Unmarshal(body, &payload)

	// Check branch
	expectedRef := "refs/heads/" + deploy.Branch
	if payload.Ref != "" && payload.Ref != expectedRef {
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf(`{"message":"skipped, branch %s != %s"}`, payload.Ref, expectedRef)))
		return
	}

	go executeDeploy(h.pool, id, payload.HeadCommit.Message, payload.After)

	w.WriteHeader(200)
	w.Write([]byte(`{"message":"deploy triggered"}`))
}

func executeDeploy(pool *pgxpool.Pool, deployID int64, commitMsg, commitHash string) {
	ctx := context.Background()
	start := time.Now()

	var d GitDeploy
	pool.QueryRow(ctx,
		"SELECT id, repo_url, branch, build_command, deploy_path FROM git_deploys WHERE id = $1", deployID,
	).Scan(&d.ID, &d.RepoURL, &d.Branch, &d.BuildCommand, &d.DeployPath)

	pool.Exec(ctx, "UPDATE git_deploys SET last_status = 'deploying' WHERE id = $1", deployID)

	tmpDir := fmt.Sprintf("/tmp/panelox-deploy-%d", deployID)
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	var output strings.Builder

	// Clone repo
	args := []string{"clone", "--depth", "1", "--branch", d.Branch, d.RepoURL, tmpDir}
	if commitHash != "" {
		args = []string{"clone", d.RepoURL, tmpDir}
	}
	out, err := exec.Command("git", args...).CombinedOutput()
	output.WriteString(string(out))
	if err != nil {
		saveDeploy(pool, ctx, deployID, commitHash, commitMsg, "failed", output.String(), time.Since(start))
		return
	}

	// Checkout specific commit if rollback
	if commitHash != "" {
		out, _ = exec.Command("git", "-C", tmpDir, "checkout", commitHash).CombinedOutput()
		output.WriteString(string(out))
	}

	// Get commit hash
	if commitHash == "" {
		hashOut, _ := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD").Output()
		commitHash = strings.TrimSpace(string(hashOut))
	}

	// Run build command
	if d.BuildCommand != "" {
		cmd := exec.Command("/bin/sh", "-c", d.BuildCommand)
		cmd.Dir = tmpDir
		out, err = cmd.CombinedOutput()
		output.WriteString(string(out))
		if err != nil {
			saveDeploy(pool, ctx, deployID, commitHash, commitMsg, "failed", output.String(), time.Since(start))
			return
		}
	}

	// Deploy: sync files to deploy path
	os.MkdirAll(d.DeployPath, 0755)
	out, err = exec.Command("rsync", "-a", "--delete",
		"--exclude", ".git",
		"--exclude", "node_modules",
		tmpDir+"/", d.DeployPath+"/").CombinedOutput()
	output.WriteString(string(out))
	if err != nil {
		// Fallback: cp
		exec.Command("cp", "-a", tmpDir+"/.", d.DeployPath+"/").Run()
	}

	saveDeploy(pool, ctx, deployID, commitHash, commitMsg, "success", output.String(), time.Since(start))
	log.Printf("Deploy %d completed: %s in %s", deployID, commitHash[:8], time.Since(start).Round(time.Millisecond))
}

func saveDeploy(pool *pgxpool.Pool, ctx context.Context, deployID int64, hash, msg, status, logOutput string, duration time.Duration) {
	// Trim log
	if len(logOutput) > 50000 {
		logOutput = logOutput[:50000] + "\n... (truncated)"
	}

	pool.Exec(ctx,
		"UPDATE git_deploys SET last_commit = $1, last_deploy = NOW(), last_status = $2, last_log = $3 WHERE id = $4",
		hash, status, logOutput, deployID)

	pool.Exec(ctx,
		`INSERT INTO deploy_history (deploy_id, commit_hash, commit_msg, status, log, duration_ms)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		deployID, hash, msg, status, logOutput, duration.Milliseconds())
}
