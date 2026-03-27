package ssl

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ACME Accounts

func (r *Repository) CreateAcmeAccount(ctx context.Context, a *AcmeAccount) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO acme_accounts (email, provider, private_key, url, key_type)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		a.Email, a.Provider, a.PrivateKey, a.URL, a.KeyType,
	).Scan(&a.ID, &a.CreatedAt)
}

func (r *Repository) GetAcmeAccount(ctx context.Context, id int64) (*AcmeAccount, error) {
	var a AcmeAccount
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, provider, private_key, url, key_type, created_at
		 FROM acme_accounts WHERE id = $1`, id,
	).Scan(&a.ID, &a.Email, &a.Provider, &a.PrivateKey, &a.URL, &a.KeyType, &a.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) ListAcmeAccounts(ctx context.Context) ([]AcmeAccount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, provider, '', url, key_type, created_at
		 FROM acme_accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []AcmeAccount
	for rows.Next() {
		var a AcmeAccount
		if err := rows.Scan(&a.ID, &a.Email, &a.Provider, &a.PrivateKey, &a.URL, &a.KeyType, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *Repository) DeleteAcmeAccount(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM acme_accounts WHERE id = $1", id)
	return err
}

// Certificates

func (r *Repository) CreateCertificate(ctx context.Context, c *Certificate) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO ssl_certificates (user_id, acme_account_id, domains, provider, key_type,
			cert_pem, key_pem, chain_pem, auto_renew, expires_at, issued_at, status,
			message, cert_url, http_config, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		 RETURNING id, created_at, updated_at`,
		c.UserID, c.AcmeAccountID, c.Domains, c.Provider, c.KeyType,
		c.CertPEM, c.KeyPEM, c.ChainPEM, c.AutoRenew, c.ExpiresAt, c.IssuedAt,
		c.Status, c.Message, c.CertURL, c.HTTPConfig, c.Description,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) GetCertificate(ctx context.Context, id int64) (*Certificate, error) {
	var c Certificate
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, acme_account_id, domains, provider, key_type,
			cert_pem, key_pem, chain_pem, auto_renew, expires_at, issued_at,
			status, message, cert_url, http_config, description, created_at, updated_at
		 FROM ssl_certificates WHERE id = $1`, id,
	).Scan(&c.ID, &c.UserID, &c.AcmeAccountID, &c.Domains, &c.Provider, &c.KeyType,
		&c.CertPEM, &c.KeyPEM, &c.ChainPEM, &c.AutoRenew, &c.ExpiresAt, &c.IssuedAt,
		&c.Status, &c.Message, &c.CertURL, &c.HTTPConfig, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListCertificates(ctx context.Context, page, size int) ([]Certificate, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM ssl_certificates").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, acme_account_id, domains, provider, key_type,
			'', '', '', auto_renew, expires_at, issued_at,
			status, message, cert_url, http_config, description, created_at, updated_at
		 FROM ssl_certificates ORDER BY id DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var certs []Certificate
	for rows.Next() {
		var c Certificate
		if err := rows.Scan(&c.ID, &c.UserID, &c.AcmeAccountID, &c.Domains, &c.Provider, &c.KeyType,
			&c.CertPEM, &c.KeyPEM, &c.ChainPEM, &c.AutoRenew, &c.ExpiresAt, &c.IssuedAt,
			&c.Status, &c.Message, &c.CertURL, &c.HTTPConfig, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		certs = append(certs, c)
	}
	return certs, total, nil
}

func (r *Repository) UpdateCertificateStatus(ctx context.Context, id int64, status, message string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE ssl_certificates SET status = $1, message = $2, updated_at = NOW() WHERE id = $3",
		status, message, id)
	return err
}

func (r *Repository) UpdateCertificateData(ctx context.Context, id int64, certPEM, keyPEM, chainPEM string, expiresAt, issuedAt time.Time, certURL string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ssl_certificates SET cert_pem = $1, key_pem = $2, chain_pem = $3,
			expires_at = $4, issued_at = $5, cert_url = $6, status = $7, message = '', updated_at = NOW()
		 WHERE id = $8`,
		certPEM, keyPEM, chainPEM, expiresAt, issuedAt, certURL, StatusIssued, id)
	return err
}

func (r *Repository) DeleteCertificate(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM ssl_certificates WHERE id = $1", id)
	return err
}

func (r *Repository) GetAutoRenewCertificates(ctx context.Context) ([]Certificate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, acme_account_id, domains, provider, key_type,
			cert_pem, key_pem, chain_pem, auto_renew, expires_at, issued_at,
			status, message, cert_url, http_config, description, created_at, updated_at
		 FROM ssl_certificates
		 WHERE auto_renew = true AND status = $1 AND provider = $2
		 AND expires_at < NOW() + INTERVAL '30 days'`,
		StatusIssued, ProviderLetsEncrypt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []Certificate
	for rows.Next() {
		var c Certificate
		if err := rows.Scan(&c.ID, &c.UserID, &c.AcmeAccountID, &c.Domains, &c.Provider, &c.KeyType,
			&c.CertPEM, &c.KeyPEM, &c.ChainPEM, &c.AutoRenew, &c.ExpiresAt, &c.IssuedAt,
			&c.Status, &c.Message, &c.CertURL, &c.HTTPConfig, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, nil
}

// Website SSL linking

func (r *Repository) ApplyToWebsite(ctx context.Context, websiteID, certID int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE websites SET ssl_enabled = true, ssl_cert_id = $1, updated_at = NOW() WHERE id = $2",
		certID, websiteID)
	return err
}

func (r *Repository) RemoveFromWebsite(ctx context.Context, websiteID int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE websites SET ssl_enabled = false, ssl_cert_id = NULL, updated_at = NOW() WHERE id = $1",
		websiteID)
	return err
}

func (r *Repository) GetWebsitesUsingCert(ctx context.Context, certID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id FROM websites WHERE ssl_cert_id = $1", certID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repository) CertificateInUse(ctx context.Context, certID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM websites WHERE ssl_cert_id = $1)", certID).Scan(&exists)
	return exists, err
}

func (r *Repository) GetWebsiteAlias(ctx context.Context, websiteID int64) (string, error) {
	var alias string
	err := r.pool.QueryRow(ctx, "SELECT alias FROM websites WHERE id = $1", websiteID).Scan(&alias)
	return alias, err
}
