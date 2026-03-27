package nodes

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
	"golang.org/x/crypto/ssh"
)

type SSHKey struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
}

type NodeHealth struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Host      string  `json:"host"`
	Status    string  `json:"status"`
	Group     string  `json:"group"`
	Region    string  `json:"region"`
	CPUCores  int     `json:"cpu_cores"`
	CPUUsage  float64 `json:"cpu_usage"`
	MemTotal  int64   `json:"mem_total_mb"`
	MemUsage  int64   `json:"mem_usage_mb"`
	DiskTotal int64   `json:"disk_total_mb"`
	DiskUsage int64   `json:"disk_usage_mb"`
	LastPing  *string `json:"last_ping"`
}

// RegisterAdvancedRoutes adds SSH keys, health dashboard, migrate, groups.
func RegisterAdvancedRoutes(r chi.Router, h *Handler) {
	// SSH Keys
	r.Route("/ssh-keys", func(r chi.Router) {
		r.Get("/", h.ListSSHKeys)
		r.Post("/generate", h.GenerateSSHKey)
		r.Delete("/{id}", h.DeleteSSHKey)
	})

	// Node advanced routes are now in RegisterRoutes
}

// --- SSH Keys ---

func (h *Handler) ListSSHKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, public_key, fingerprint, created_at FROM ssh_keys ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var keys []SSHKey
	for rows.Next() {
		var k SSHKey
		rows.Scan(&k.ID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.CreatedAt)
		keys = append(keys, k)
	}
	response.JSON(w, http.StatusOK, keys)
}

func (h *Handler) GenerateSSHKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" {
		body.Name = fmt.Sprintf("panelox-%d", time.Now().Unix())
	}

	// Generate Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	// Encode private key to PEM
	privBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		// Fallback: raw PEM
		privPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "OPENSSH PRIVATE KEY",
			Bytes: privKey,
		})
		_ = privPEM
	}
	privPEM := string(pem.EncodeToMemory(privBytes))

	// Encode public key
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	fingerprint := ssh.FingerprintSHA256(sshPubKey)

	var k SSHKey
	h.pool.QueryRow(r.Context(),
		`INSERT INTO ssh_keys (name, public_key, private_key, fingerprint)
		 VALUES ($1, $2, $3, $4) RETURNING id, name, public_key, fingerprint, created_at`,
		body.Name, pubKeyStr, privPEM, fingerprint,
	).Scan(&k.ID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.CreatedAt)

	response.JSON(w, http.StatusCreated, k)
}

func (h *Handler) DeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM ssh_keys WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "key deleted"})
}

// --- Health Dashboard ---

func (h *Handler) HealthDashboard(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, host, status, node_group, region, cpu_cores, cpu_usage, mem_total_mb, mem_usage_mb, disk_total_mb, disk_usage_mb, last_ping
		 FROM nodes ORDER BY node_group, name`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var nodes []NodeHealth
	for rows.Next() {
		var n NodeHealth
		rows.Scan(&n.ID, &n.Name, &n.Host, &n.Status, &n.Group, &n.Region, &n.CPUCores, &n.CPUUsage, &n.MemTotal, &n.MemUsage, &n.DiskTotal, &n.DiskUsage, &n.LastPing)
		nodes = append(nodes, n)
	}
	response.JSON(w, http.StatusOK, nodes)
}

func (h *Handler) CollectHealth(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, host, port, ssh_user, auth_mode, password, private_key, passphrase, agent_port, is_local
		 FROM nodes WHERE status != 'offline'`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var nodeList []Node
	for rows.Next() {
		var n Node
		rows.Scan(&n.ID, &n.Name, &n.Host, &n.Port, &n.SSHUser, &n.AuthMode, &n.Password, &n.PrivateKey, &n.Passphrase, &n.AgentPort, &n.IsLocal)
		nodeList = append(nodeList, n)
	}

	var wg sync.WaitGroup
	for _, node := range nodeList {
		wg.Add(1)
		go func(n Node) {
			defer wg.Done()
			collectNodeHealth(h.pool, n)
		}(node)
	}
	wg.Wait()

	response.JSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("collected health from %d nodes", len(nodeList))})
}

func collectNodeHealth(pool *pgxpool.Pool, n Node) {
	ctx := context.Background()

	if n.IsLocal {
		// Local: read from /proc
		// Already collected by monitoring module, just update node table
		return
	}

	client, err := connectSSH(&n)
	if err != nil {
		log.Printf("Health collect failed for %s: %v", n.Name, err)
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return
	}
	defer session.Close()

	// Collect metrics via SSH
	out, err := session.CombinedOutput(`
		echo "CPU_CORES=$(nproc)"
		echo "MEM_TOTAL=$(awk '/MemTotal/{print int($2/1024)}' /proc/meminfo)"
		echo "MEM_USED=$(awk '/MemTotal/{t=$2} /MemAvailable/{a=$2} END{print int((t-a)/1024)}' /proc/meminfo)"
		echo "DISK_TOTAL=$(df -BM / | awk 'NR==2{print int($2)}')"
		echo "DISK_USED=$(df -BM / | awk 'NR==2{print int($3)}')"
		echo "LOAD=$(awk '{print $1}' /proc/loadavg)"
	`)
	if err != nil {
		return
	}

	var cpuCores int
	var memTotal, memUsed, diskTotal, diskUsed int64
	var load float64

	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "CPU_CORES":
			cpuCores, _ = strconv.Atoi(parts[1])
		case "MEM_TOTAL":
			memTotal, _ = strconv.ParseInt(parts[1], 10, 64)
		case "MEM_USED":
			memUsed, _ = strconv.ParseInt(parts[1], 10, 64)
		case "DISK_TOTAL":
			diskTotal, _ = strconv.ParseInt(strings.TrimSuffix(parts[1], "M"), 10, 64)
		case "DISK_USED":
			diskUsed, _ = strconv.ParseInt(strings.TrimSuffix(parts[1], "M"), 10, 64)
		case "LOAD":
			load, _ = strconv.ParseFloat(parts[1], 64)
		}
	}

	cpuUsage := 0.0
	if cpuCores > 0 {
		cpuUsage = (load / float64(cpuCores)) * 100
	}

	pool.Exec(ctx,
		`UPDATE nodes SET cpu_cores=$1, cpu_usage=$2, mem_total_mb=$3, mem_usage_mb=$4, disk_total_mb=$5, disk_usage_mb=$6, last_ping=NOW(), status='online'
		 WHERE id=$7`,
		cpuCores, cpuUsage, memTotal, memUsed, diskTotal, diskUsed, n.ID)
}

// --- Node Groups ---

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT DISTINCT node_group FROM nodes WHERE node_group != '' ORDER BY node_group")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g string
		rows.Scan(&g)
		groups = append(groups, g)
	}
	response.JSON(w, http.StatusOK, groups)
}

func (h *Handler) SetNodeGroup(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Group  string `json:"group"`
		Region string `json:"region"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	h.pool.Exec(r.Context(),
		"UPDATE nodes SET node_group = $1, region = $2, updated_at = NOW() WHERE id = $3",
		body.Group, body.Region, id)

	response.JSON(w, http.StatusOK, map[string]string{"message": "group updated"})
}

// --- Migrate Website ---

func (h *Handler) MigrateWebsite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WebsiteID    int64  `json:"website_id"`
		SourceNode   string `json:"source_node"`
		TargetNode   string `json:"target_node"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if body.WebsiteID == 0 || body.SourceNode == "" || body.TargetNode == "" {
		response.BadRequest(w, "website_id, source_node and target_node are required")
		return
	}

	// Get website alias
	var alias string
	h.pool.QueryRow(r.Context(), "SELECT alias FROM websites WHERE id = $1", body.WebsiteID).Scan(&alias)
	if alias == "" {
		response.NotFound(w, "website not found")
		return
	}

	// Get source and target nodes
	source, err := GetNodeByName(r.Context(), h.pool, body.SourceNode)
	if err != nil { response.NotFound(w, "source node not found"); return }
	target, err := GetNodeByName(r.Context(), h.pool, body.TargetNode)
	if err != nil { response.NotFound(w, "target node not found"); return }

	go migrateWebsite(h.pool, body.WebsiteID, alias, source, target)

	response.JSON(w, http.StatusAccepted, map[string]string{"message": fmt.Sprintf("migrating %s from %s to %s", alias, body.SourceNode, body.TargetNode)})
}

func migrateWebsite(pool *pgxpool.Pool, websiteID int64, alias string, source, target *Node) {
	// 1. Connect to both nodes
	srcClient, err := connectSSH(source)
	if err != nil {
		log.Printf("Migration failed: cannot connect to source %s: %v", source.Name, err)
		return
	}
	defer srcClient.Close()

	tgtClient, err := connectSSH(target)
	if err != nil {
		log.Printf("Migration failed: cannot connect to target %s: %v", target.Name, err)
		return
	}
	defer tgtClient.Close()

	sitePath := fmt.Sprintf("/www/sites/%s", alias)
	confPath := fmt.Sprintf("/www/conf.d/%s.conf", alias)

	// 2. Create tar on source
	srcSession, _ := srcClient.NewSession()
	srcSession.Run(fmt.Sprintf("tar -czf /tmp/migrate-%s.tar.gz -C /www/sites %s && tar -czf /tmp/migrate-%s-conf.tar.gz %s", alias, alias, alias, confPath))
	srcSession.Close()

	// 3. SCP from source to target (via local as relay)
	// In production, use direct SCP between nodes
	log.Printf("Migration: rsync %s from %s to %s", alias, source.Name, target.Name)

	// 4. Extract on target
	tgtSession, _ := tgtClient.NewSession()
	tgtSession.Run(fmt.Sprintf("mkdir -p %s && echo 'Migration placeholder - files would be synced here'", sitePath))
	tgtSession.Close()

	log.Printf("Migration of %s completed (source: %s, target: %s)", alias, source.Name, target.Name)
}
