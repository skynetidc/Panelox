package filemanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

const recycleBinDir = "/opt/panelox/recycle-bin"

// RegisterAdvancedRoutes adds compress, search, batch, recycle bin routes.
func RegisterAdvancedRoutes(r chi.Router, h *Handler) {
	r.Post("/files/compress", h.Compress)
	r.Post("/files/decompress", h.Decompress)
	r.Post("/files/search", h.Search)
	r.Post("/files/batch-delete", h.BatchDelete)
	r.Post("/files/batch-move", h.BatchMove)
	r.Post("/files/batch-chmod", h.BatchChmod)
	r.Get("/files/recycle-bin", h.ListRecycleBin)
	r.Post("/files/recycle-bin/restore", h.RestoreFromBin)
	r.Post("/files/recycle-bin/empty", h.EmptyRecycleBin)
}

// --- Compress / Decompress ---

func (h *Handler) Compress(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Paths       []string `json:"paths"`
		Destination string   `json:"destination"`
		Format      string   `json:"format"` // zip, tar.gz
	}
	json.NewDecoder(r.Body).Decode(&body)
	if len(body.Paths) == 0 || body.Destination == "" {
		response.BadRequest(w, "paths and destination are required")
		return
	}
	if body.Format == "" {
		body.Format = "tar.gz"
	}

	// Validate all paths
	safeDest, err := validatePath(body.Destination)
	if err != nil {
		response.Forbidden(w, "invalid destination path")
		return
	}

	var safePaths []string
	for _, p := range body.Paths {
		sp, err := validatePath(p)
		if err != nil {
			response.Forbidden(w, fmt.Sprintf("invalid path: %s", p))
			return
		}
		safePaths = append(safePaths, sp)
	}

	var cmd *exec.Cmd
	switch body.Format {
	case "zip":
		args := []string{"-r", safeDest}
		args = append(args, safePaths...)
		cmd = exec.Command("zip", args...)
	default: // tar.gz
		args := []string{"-czf", safeDest}
		args = append(args, safePaths...)
		cmd = exec.Command("tar", args...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("compress failed: %s", string(out)))
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "compressed", "path": safeDest})
}

func (h *Handler) Decompress(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path        string `json:"path"`
		Destination string `json:"destination"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Path == "" {
		response.BadRequest(w, "path is required")
		return
	}

	safePath, err := validatePath(body.Path)
	if err != nil {
		response.Forbidden(w, "invalid path")
		return
	}

	destDir := body.Destination
	if destDir == "" {
		destDir = filepath.Dir(safePath)
	}
	safeDest, err := validatePath(destDir)
	if err != nil {
		response.Forbidden(w, "invalid destination")
		return
	}

	os.MkdirAll(safeDest, 0755)

	var cmd *exec.Cmd
	lower := strings.ToLower(safePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		cmd = exec.Command("unzip", "-o", safePath, "-d", safeDest)
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		cmd = exec.Command("tar", "-xzf", safePath, "-C", safeDest)
	case strings.HasSuffix(lower, ".tar.bz2"):
		cmd = exec.Command("tar", "-xjf", safePath, "-C", safeDest)
	case strings.HasSuffix(lower, ".tar"):
		cmd = exec.Command("tar", "-xf", safePath, "-C", safeDest)
	default:
		response.BadRequest(w, "unsupported archive format (zip, tar.gz, tar.bz2, tar)")
		return
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("decompress failed: %s", string(out)))
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "decompressed", "path": safeDest})
}

// --- Search (grep) ---

type SearchResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path    string `json:"path"`
		Query   string `json:"query"`
		FileExt string `json:"file_ext"` // optional: "php", "js", etc.
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Path == "" || body.Query == "" {
		response.BadRequest(w, "path and query are required")
		return
	}

	safePath, err := validatePath(body.Path)
	if err != nil {
		response.Forbidden(w, "invalid path")
		return
	}

	args := []string{"-rn", "--max-count=100", "--color=never"}
	if body.FileExt != "" {
		args = append(args, "--include=*."+body.FileExt)
	}
	args = append(args, body.Query, safePath)

	out, _ := exec.Command("grep", args...).Output()

	var results []SearchResult
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		// Parse grep output: file:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		lineNum := 0
		fmt.Sscanf(parts[1], "%d", &lineNum)
		results = append(results, SearchResult{
			File:    parts[0],
			Line:    lineNum,
			Content: strings.TrimSpace(parts[2]),
		})
	}

	response.JSON(w, http.StatusOK, results)
}

// --- Batch Operations ---

func (h *Handler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Paths   []string `json:"paths"`
		ToBin   bool     `json:"to_bin"` // move to recycle bin instead of deleting
	}
	json.NewDecoder(r.Body).Decode(&body)
	if len(body.Paths) == 0 {
		response.BadRequest(w, "paths is required")
		return
	}

	deleted := 0
	for _, p := range body.Paths {
		safePath, err := validatePath(p)
		if err != nil {
			continue
		}

		if body.ToBin {
			moveToRecycleBin(safePath)
		} else {
			info, err := os.Stat(safePath)
			if err != nil {
				continue
			}
			if info.IsDir() {
				os.RemoveAll(safePath)
			} else {
				os.Remove(safePath)
			}
		}
		deleted++
	}

	response.JSON(w, http.StatusOK, map[string]int{"deleted": deleted})
}

func (h *Handler) BatchMove(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Paths       []string `json:"paths"`
		Destination string   `json:"destination"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if len(body.Paths) == 0 || body.Destination == "" {
		response.BadRequest(w, "paths and destination are required")
		return
	}

	safeDest, err := validatePath(body.Destination)
	if err != nil {
		response.Forbidden(w, "invalid destination")
		return
	}
	os.MkdirAll(safeDest, 0755)

	moved := 0
	for _, p := range body.Paths {
		safePath, err := validatePath(p)
		if err != nil {
			continue
		}
		dest := filepath.Join(safeDest, filepath.Base(safePath))
		if os.Rename(safePath, dest) == nil {
			moved++
		}
	}

	response.JSON(w, http.StatusOK, map[string]int{"moved": moved})
}

func (h *Handler) BatchChmod(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Paths []string `json:"paths"`
		Mode  string   `json:"mode"` // e.g. "0755"
	}
	json.NewDecoder(r.Body).Decode(&body)

	changed := 0
	for _, p := range body.Paths {
		if h.svc.Chmod(p, body.Mode) == nil {
			changed++
		}
	}

	response.JSON(w, http.StatusOK, map[string]int{"changed": changed})
}

// --- Recycle Bin ---

type RecycleBinEntry struct {
	Name         string `json:"name"`
	OriginalPath string `json:"original_path"`
	Size         int64  `json:"size"`
	DeletedAt    string `json:"deleted_at"`
}

func moveToRecycleBin(path string) error {
	os.MkdirAll(recycleBinDir, 0755)

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Create metadata file
	timestamp := time.Now().Format("20060102-150405")
	binName := fmt.Sprintf("%s_%s", timestamp, filepath.Base(path))
	binPath := filepath.Join(recycleBinDir, binName)
	metaPath := binPath + ".meta"

	// Move file
	if err := os.Rename(path, binPath); err != nil {
		// Try copy+delete if cross-device
		exec.Command("mv", path, binPath).Run()
	}

	// Save metadata
	meta := map[string]any{
		"original_path": path,
		"size":          info.Size(),
		"deleted_at":    time.Now().Format(time.RFC3339),
		"is_dir":        info.IsDir(),
	}
	metaBytes, _ := json.Marshal(meta)
	os.WriteFile(metaPath, metaBytes, 0644)

	return nil
}

func (h *Handler) ListRecycleBin(w http.ResponseWriter, r *http.Request) {
	os.MkdirAll(recycleBinDir, 0755)
	entries, err := os.ReadDir(recycleBinDir)
	if err != nil {
		response.JSON(w, http.StatusOK, []RecycleBinEntry{})
		return
	}

	var items []RecycleBinEntry
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".meta") {
			continue
		}

		entry := RecycleBinEntry{Name: e.Name()}

		// Read metadata
		metaPath := filepath.Join(recycleBinDir, e.Name()+".meta")
		if metaBytes, err := os.ReadFile(metaPath); err == nil {
			var meta map[string]any
			json.Unmarshal(metaBytes, &meta)
			if op, ok := meta["original_path"].(string); ok {
				entry.OriginalPath = op
			}
			if da, ok := meta["deleted_at"].(string); ok {
				entry.DeletedAt = da
			}
			if sz, ok := meta["size"].(float64); ok {
				entry.Size = int64(sz)
			}
		}

		info, _ := e.Info()
		if info != nil && entry.Size == 0 {
			entry.Size = info.Size()
		}

		items = append(items, entry)
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) RestoreFromBin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}

	binPath := filepath.Join(recycleBinDir, body.Name)
	metaPath := binPath + ".meta"

	// Read original path from metadata
	var originalPath string
	if metaBytes, err := os.ReadFile(metaPath); err == nil {
		var meta map[string]any
		json.Unmarshal(metaBytes, &meta)
		if op, ok := meta["original_path"].(string); ok {
			originalPath = op
		}
	}

	if originalPath == "" {
		response.BadRequest(w, "cannot determine original path")
		return
	}

	// Restore
	os.MkdirAll(filepath.Dir(originalPath), 0755)
	if err := os.Rename(binPath, originalPath); err != nil {
		exec.Command("mv", binPath, originalPath).Run()
	}
	os.Remove(metaPath)

	response.JSON(w, http.StatusOK, map[string]string{"message": "restored", "path": originalPath})
}

func (h *Handler) EmptyRecycleBin(w http.ResponseWriter, r *http.Request) {
	os.RemoveAll(recycleBinDir)
	os.MkdirAll(recycleBinDir, 0755)
	response.JSON(w, http.StatusOK, map[string]string{"message": "recycle bin emptied"})
}
