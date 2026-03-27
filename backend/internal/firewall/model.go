package firewall

import "time"

type Rule struct {
	ID          int64     `json:"id"`
	Chain       string    `json:"chain"`
	Protocol    string    `json:"protocol"`
	SrcIP       string    `json:"src_ip,omitempty"`
	DstPort     string    `json:"dst_port"`
	Action      string    `json:"action"`
	Description string    `json:"description,omitempty"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateRuleRequest struct {
	Chain       string `json:"chain"`
	Protocol    string `json:"protocol"`
	SrcIP       string `json:"src_ip"`
	DstPort     string `json:"dst_port"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type StatusResponse struct {
	Enabled bool   `json:"enabled"`
	Backend string `json:"backend"` // iptables or nftables
}
