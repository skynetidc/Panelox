package mail

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrDomainTaken = errors.New("domain already exists")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Domains

func (s *Service) CreateDomain(ctx context.Context, userID int64, req CreateDomainRequest) (*Domain, error) {
	// Check uniqueness
	var exists bool
	s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM mail_domains WHERE domain = $1)", req.Domain).Scan(&exists)
	if exists {
		return nil, ErrDomainTaken
	}

	d := &Domain{UserID: userID, Domain: req.Domain, Status: "active"}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO mail_domains (user_id, domain, status) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		d.UserID, d.Domain, d.Status,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) ListDomains(ctx context.Context, page, size int) ([]Domain, int64, error) {
	var total int64
	s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM mail_domains").Scan(&total)
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }
	offset := (page - 1) * size

	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, domain, COALESCE(container_id,''), COALESCE(dkim_public_key,''), status, created_at, updated_at
		 FROM mail_domains ORDER BY id DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.UserID, &d.Domain, &d.ContainerID, &d.DKIMPublicKey, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		domains = append(domains, d)
	}
	return domains, total, nil
}

func (s *Service) DeleteDomain(ctx context.Context, id int64) error {
	ct, err := s.pool.Exec(ctx, "DELETE FROM mail_domains WHERE id = $1", id)
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// Mailboxes

func (s *Service) CreateMailbox(ctx context.Context, domainID int64, req CreateMailboxRequest) (*Mailbox, error) {
	// Verify domain exists
	var exists bool
	s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM mail_domains WHERE id = $1)", domainID).Scan(&exists)
	if !exists {
		return nil, ErrNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, err
	}

	quota := req.QuotaMB
	if quota <= 0 {
		quota = 1024
	}

	m := &Mailbox{MailDomainID: domainID, Address: req.Address, Password: string(hash), QuotaMB: quota, Status: "active"}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO mailboxes (mail_domain_id, address, password, quota_mb, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		m.MailDomainID, m.Address, m.Password, m.QuotaMB, m.Status,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) ListMailboxes(ctx context.Context, domainID int64) ([]Mailbox, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, mail_domain_id, address, '', quota_mb, status, created_at, updated_at
		 FROM mailboxes WHERE mail_domain_id = $1 ORDER BY id`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mailboxes []Mailbox
	for rows.Next() {
		var m Mailbox
		if err := rows.Scan(&m.ID, &m.MailDomainID, &m.Address, &m.Password, &m.QuotaMB, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		mailboxes = append(mailboxes, m)
	}
	return mailboxes, nil
}

func (s *Service) DeleteMailbox(ctx context.Context, id int64) error {
	var dummy int64
	err := s.pool.QueryRow(ctx, "SELECT id FROM mailboxes WHERE id = $1", id).Scan(&dummy)
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}
	_, err = s.pool.Exec(ctx, "DELETE FROM mailboxes WHERE id = $1", id)
	return err
}
