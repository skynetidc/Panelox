package mail

import "time"

type Domain struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Domain        string    `json:"domain"`
	ContainerID   string    `json:"container_id,omitempty"`
	DKIMPublicKey string    `json:"dkim_public_key,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Mailbox struct {
	ID           int64     `json:"id"`
	MailDomainID int64     `json:"mail_domain_id"`
	Address      string    `json:"address"`
	Password     string    `json:"-"`
	QuotaMB      int       `json:"quota_mb"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateDomainRequest struct {
	Domain string `json:"domain"`
}

type CreateMailboxRequest struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	QuotaMB  int    `json:"quota_mb"`
}
