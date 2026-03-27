package ssl

import "time"

// Provider types
const (
	ProviderLetsEncrypt = "letsencrypt"
	ProviderCustom      = "custom"
)

// Certificate statuses
const (
	StatusPending  = "pending"
	StatusIssuing  = "issuing"
	StatusIssued   = "issued"
	StatusExpired  = "expired"
	StatusFailed   = "failed"
)

// HTTP config options (how to handle HTTP when SSL is enabled)
const (
	HTTPToHTTPS = "HTTPToHTTPS" // Redirect HTTP to HTTPS
	HTTPSOnly   = "HTTPSOnly"   // Only serve HTTPS
	HTTPAlso    = "HTTPAlso"    // Serve both HTTP and HTTPS
)

type Certificate struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	AcmeAccountID *int64     `json:"acme_account_id,omitempty"`
	Domains       []string   `json:"domains"`
	Provider      string     `json:"provider"`
	KeyType       string     `json:"key_type"`
	CertPEM       string     `json:"cert_pem,omitempty"`
	KeyPEM        string     `json:"-"`
	ChainPEM      string     `json:"chain_pem,omitempty"`
	AutoRenew     bool       `json:"auto_renew"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	IssuedAt      *time.Time `json:"issued_at,omitempty"`
	Status        string     `json:"status"`
	Message       string     `json:"message,omitempty"`
	CertURL       string     `json:"cert_url,omitempty"`
	HTTPConfig    string     `json:"http_config"`
	Description   string     `json:"description,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AcmeAccount struct {
	ID         int64     `json:"id"`
	Email      string    `json:"email"`
	Provider   string    `json:"provider"`
	PrivateKey string    `json:"-"`
	URL        string    `json:"url,omitempty"`
	KeyType    string    `json:"key_type"`
	CreatedAt  time.Time `json:"created_at"`
}

// Request DTOs

type CreateAcmeAccountRequest struct {
	Email    string `json:"email"`
	Provider string `json:"provider"`
	KeyType  string `json:"key_type"`
}

type IssueRequest struct {
	Domains       []string `json:"domains"`
	AcmeAccountID int64    `json:"acme_account_id"`
	KeyType       string   `json:"key_type"`
	AutoRenew     bool     `json:"auto_renew"`
	HTTPConfig    string   `json:"http_config"`
	Description   string   `json:"description"`
}

type UploadRequest struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	Description string `json:"description"`
}

type RenewRequest struct {
	ID int64 `json:"id"`
}

type ApplyRequest struct {
	CertificateID int64  `json:"certificate_id"`
	WebsiteID     int64  `json:"website_id"`
	HTTPConfig    string `json:"http_config"`
}
