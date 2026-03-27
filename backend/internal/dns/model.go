package dns

import "time"

type Zone struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Serial    int64     `json:"serial"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Records   []Record  `json:"records,omitempty"`
}

type Record struct {
	ID        int64     `json:"id"`
	ZoneID    int64     `json:"zone_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	TTL       int       `json:"ttl"`
	Priority  *int      `json:"priority,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Record types
var ValidRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT", "SRV", "NS", "CAA", "PTR"}

// Request DTOs

type CreateZoneRequest struct {
	Name string `json:"name"`
}

type CreateRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
}

type UpdateRecordRequest struct {
	Name     *string `json:"name,omitempty"`
	Type     *string `json:"type,omitempty"`
	Content  *string `json:"content,omitempty"`
	TTL      *int    `json:"ttl,omitempty"`
	Priority *int    `json:"priority,omitempty"`
}
