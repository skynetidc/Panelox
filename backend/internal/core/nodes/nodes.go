package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
	"golang.org/x/crypto/ssh"
)

type Node struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	SSHUser    string     `json:"ssh_user"`
	AuthMode   string     `json:"auth_mode"`
	Password   string     `json:"-"`
	PrivateKey string     `json:"-"`
	Passphrase string     `json:"-"`
	AgentPort  int        `json:"agent_port"`
	Status     string     `json:"status"`
	IsLocal    bool       `json:"is_local"`
	NodeGroup  string     `json:"node_group"`
	Region     string     `json:"region"`
	LastPing   *time.Time `json:"last_ping,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateNodeRequest struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	SSHUser    string `json:"ssh_user"`
	AuthMode   string `json:"auth_mode"` // password or key
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	AgentPort  int    `json:"agent_port"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/nodes", h.List)
	r.Post("/nodes", h.Create)
	r.Post("/nodes/ping-all", h.PingAll)
	r.Get("/nodes/health", h.HealthDashboard)
	r.Get("/nodes/groups", h.ListGroups)
	r.Post("/nodes/migrate", h.MigrateWebsite)
	r.Post("/nodes/collect-health", h.CollectHealth)
	r.Route("/nodes/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Delete("/", h.Delete)
		r.Post("/test", h.TestConnection)
		r.Post("/install-agent", h.InstallAgent)
		r.Put("/group", h.SetNodeGroup)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, host, port, ssh_user, auth_mode, '', '', '', agent_port, status, is_local, COALESCE(node_group,'default'), COALESCE(region,''), last_ping, created_at, updated_at
		 FROM nodes ORDER BY is_local DESC, id`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		rows.Scan(&n.ID, &n.Name, &n.Host, &n.Port, &n.SSHUser, &n.AuthMode, &n.Password, &n.PrivateKey, &n.Passphrase, &n.AgentPort, &n.Status, &n.IsLocal, &n.NodeGroup, &n.Region, &n.LastPing, &n.CreatedAt, &n.UpdatedAt)
		nodes = append(nodes, n)
	}
	response.JSON(w, http.StatusOK, nodes)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	node, err := getNode(r.Context(), h.pool, id)
	if err != nil {
		response.NotFound(w, "node not found")
		return
	}
	response.JSON(w, http.StatusOK, node)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Host == "" {
		response.BadRequest(w, "name and host are required")
		return
	}
	if req.Port == 0 { req.Port = 22 }
	if req.SSHUser == "" { req.SSHUser = "root" }
	if req.AgentPort == 0 { req.AgentPort = 9090 }
	if req.AuthMode == "" { req.AuthMode = "password" }

	var n Node
	err := h.pool.QueryRow(r.Context(),
		`INSERT INTO nodes (name, host, port, ssh_user, auth_mode, password, private_key, passphrase, agent_port, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'offline')
		 RETURNING id, name, host, port, ssh_user, auth_mode, agent_port, status, is_local, last_ping, created_at, updated_at`,
		req.Name, req.Host, req.Port, req.SSHUser, req.AuthMode, req.Password, req.PrivateKey, req.Passphrase, req.AgentPort,
	).Scan(&n.ID, &n.Name, &n.Host, &n.Port, &n.SSHUser, &n.AuthMode, &n.AgentPort, &n.Status, &n.IsLocal, &n.LastPing, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, n)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	// Prevent deleting local node
	var isLocal bool
	h.pool.QueryRow(r.Context(), "SELECT is_local FROM nodes WHERE id = $1", id).Scan(&isLocal)
	if isLocal {
		response.BadRequest(w, "cannot delete local node")
		return
	}

	h.pool.Exec(r.Context(), "DELETE FROM nodes WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "node deleted"})
}

func (h *Handler) TestConnection(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	node, err := getNodeWithCreds(r.Context(), h.pool, id)
	if err != nil {
		response.NotFound(w, "node not found")
		return
	}

	if node.IsLocal {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "local node"})
		return
	}

	client, err := connectSSH(node)
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
		return
	}
	client.Close()

	h.pool.Exec(r.Context(), "UPDATE nodes SET status = 'online', last_ping = NOW() WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "SSH connection successful"})
}

func (h *Handler) InstallAgent(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	node, err := getNodeWithCreds(r.Context(), h.pool, id)
	if err != nil {
		response.NotFound(w, "node not found")
		return
	}

	if node.IsLocal {
		response.BadRequest(w, "agent is already running locally")
		return
	}

	client, err := connectSSH(node)
	if err != nil {
		response.InternalError(w, fmt.Sprintf("SSH connection failed: %v", err))
		return
	}
	defer client.Close()

	// Run agent installation commands via SSH
	installScript := `
set -e
mkdir -p /opt/panelox/bin /opt/panelox/migrations /www/sites /www/conf.d
# Download agent binary (placeholder - in production would download from release)
echo "Agent directory prepared at /opt/panelox"
# Create systemd service for agent
cat > /etc/systemd/system/panelox-agent.service <<'EOF'
[Unit]
Description=Panelox Agent
After=network.target docker.service

[Service]
Type=simple
ExecStart=/opt/panelox/bin/panelox-agent
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
echo "Agent service created. Upload binary to /opt/panelox/bin/panelox-agent and run: systemctl start panelox-agent"
`

	session, err := client.NewSession()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("SSH session failed: %v", err))
		return
	}
	defer session.Close()

	output, err := session.CombinedOutput(installScript)
	if err != nil {
		response.InternalError(w, fmt.Sprintf("Install failed: %s", string(output)))
		return
	}

	h.pool.Exec(r.Context(), "UPDATE nodes SET status = 'installed', updated_at = NOW() WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": string(output)})
}

func (h *Handler) PingAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, host, port, ssh_user, auth_mode, password, private_key, passphrase, agent_port, is_local
		 FROM nodes`)
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
	results := make(map[string]string)
	var mu sync.Mutex

	for _, node := range nodeList {
		wg.Add(1)
		go func(n Node) {
			defer wg.Done()
			status := "online"

			if n.IsLocal {
				// Check local agent via Unix socket or TCP
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", n.AgentPort))
				if err != nil || resp.StatusCode != 200 {
					status = "offline"
				}
			} else {
				sshClient, err := connectSSH(&n)
				if err != nil {
					status = "offline"
				} else {
					sshClient.Close()
				}
			}

			mu.Lock()
			results[n.Name] = status
			mu.Unlock()

			h.pool.Exec(context.Background(),
				"UPDATE nodes SET status = $1, last_ping = NOW() WHERE id = $2", status, n.ID)
		}(node)
	}

	wg.Wait()
	response.JSON(w, http.StatusOK, results)
}

// SSH helpers

func connectSSH(node *Node) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            node.SSHUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	switch node.AuthMode {
	case "password":
		config.Auth = []ssh.AuthMethod{ssh.Password(node.Password)}
	case "key":
		var signer ssh.Signer
		var err error
		if node.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(node.PrivateKey), []byte(node.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(node.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("parse SSH key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}

	addr := fmt.Sprintf("%s:%d", node.Host, node.Port)
	return ssh.Dial("tcp", addr, config)
}

func getNode(ctx context.Context, pool *pgxpool.Pool, id int64) (*Node, error) {
	var n Node
	err := pool.QueryRow(ctx,
		`SELECT id, name, host, port, ssh_user, auth_mode, agent_port, status, is_local, COALESCE(node_group,'default'), COALESCE(region,''), last_ping, created_at, updated_at
		 FROM nodes WHERE id = $1`, id,
	).Scan(&n.ID, &n.Name, &n.Host, &n.Port, &n.SSHUser, &n.AuthMode, &n.AgentPort, &n.Status, &n.IsLocal, &n.NodeGroup, &n.Region, &n.LastPing, &n.CreatedAt, &n.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("not found")
	}
	return &n, err
}

func getNodeWithCreds(ctx context.Context, pool *pgxpool.Pool, id int64) (*Node, error) {
	var n Node
	err := pool.QueryRow(ctx,
		`SELECT id, name, host, port, ssh_user, auth_mode, COALESCE(password,''), COALESCE(private_key,''), COALESCE(passphrase,''), agent_port, status, is_local
		 FROM nodes WHERE id = $1`, id,
	).Scan(&n.ID, &n.Name, &n.Host, &n.Port, &n.SSHUser, &n.AuthMode, &n.Password, &n.PrivateKey, &n.Passphrase, &n.AgentPort, &n.Status, &n.IsLocal)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("not found")
	}
	return &n, err
}

// GetNodeByName returns a node by name (used by core proxy).
func GetNodeByName(ctx context.Context, pool *pgxpool.Pool, name string) (*Node, error) {
	var n Node
	err := pool.QueryRow(ctx,
		`SELECT id, name, host, port, ssh_user, auth_mode, password, private_key, passphrase, agent_port, status, is_local
		 FROM nodes WHERE name = $1`, name,
	).Scan(&n.ID, &n.Name, &n.Host, &n.Port, &n.SSHUser, &n.AuthMode, &n.Password, &n.PrivateKey, &n.Passphrase, &n.AgentPort, &n.Status, &n.IsLocal)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("not found")
	}
	return &n, err
}

// ConnectToNode establishes an SSH connection to a node (for proxy use).
func ConnectToNode(node *Node) (*ssh.Client, error) {
	return connectSSH(node)
}
