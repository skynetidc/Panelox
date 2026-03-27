package website

import "time"

// Website types
const (
	TypeStatic  = "static"
	TypeProxy   = "proxy"
	TypePHP     = "php"
	TypeNode    = "node"
	TypeRuntime = "runtime"
)

// Website statuses
const (
	StatusActive  = "active"
	StatusStopped = "stopped"
)

type Website struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Domain        string    `json:"domain"`
	Alias         string    `json:"alias"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	DocumentRoot  string    `json:"document_root"`
	ProxyTarget   string    `json:"proxy_target,omitempty"`
	ContainerID   string    `json:"container_id,omitempty"`
	PHPVersion    string    `json:"php_version,omitempty"`
	SSLEnabled    bool      `json:"ssl_enabled"`
	SSLCertID     *int64    `json:"ssl_cert_id,omitempty"`
	IPV6          bool      `json:"ipv6"`
	AccessLog     bool      `json:"access_log"`
	ErrorLog      bool      `json:"error_log"`
	DefaultServer bool      `json:"default_server"`
	NginxCustom   string    `json:"nginx_custom,omitempty"`
	Remark        string    `json:"remark,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Joined data (not always populated)
	Domains []WebsiteDomain `json:"domains,omitempty"`
}

type WebsiteDomain struct {
	ID        int64     `json:"id"`
	WebsiteID int64     `json:"website_id"`
	Domain    string    `json:"domain"`
	Port      int       `json:"port"`
	SSL       bool      `json:"ssl"`
	CreatedAt time.Time `json:"created_at"`
}

// Request DTOs

type CreateRequest struct {
	Domain      string          `json:"domain"`
	Alias       string          `json:"alias"`
	Type        string          `json:"type"`
	ProxyTarget string          `json:"proxy_target,omitempty"`
	PHPVersion  string          `json:"php_version,omitempty"`
	IPV6        bool            `json:"ipv6"`
	Remark      string          `json:"remark,omitempty"`
	Domains     []DomainEntry   `json:"domains,omitempty"`
}

type DomainEntry struct {
	Domain string `json:"domain"`
	Port   int    `json:"port"`
}

type UpdateRequest struct {
	Domain        *string `json:"domain,omitempty"`
	ProxyTarget   *string `json:"proxy_target,omitempty"`
	PHPVersion    *string `json:"php_version,omitempty"`
	IPV6          *bool   `json:"ipv6,omitempty"`
	DefaultServer *bool   `json:"default_server,omitempty"`
	AccessLog     *bool   `json:"access_log,omitempty"`
	ErrorLog      *bool   `json:"error_log,omitempty"`
	Remark        *string `json:"remark,omitempty"`
}

type ListQuery struct {
	Page   int    `json:"page"`
	Size   int    `json:"size"`
	Search string `json:"search"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type NginxConfigUpdate struct {
	Config string `json:"config"`
}

type OperateRequest struct {
	Action string `json:"action"` // start, stop
}

type DomainCreateRequest struct {
	Domain string `json:"domain"`
	Port   int    `json:"port"`
}
