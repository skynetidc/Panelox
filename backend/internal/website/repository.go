package website

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const websiteCols = `id, user_id, domain, COALESCE(alias,''), type, status,
	COALESCE(document_root,''), COALESCE(proxy_target,''), COALESCE(container_id,''),
	COALESCE(php_version,''), ssl_enabled, ssl_cert_id, ipv6, access_log, error_log,
	default_server, COALESCE(nginx_custom,''), remark, created_at, updated_at`

func scanWebsite(row pgx.Row) (*Website, error) {
	var w Website
	var aliases string
	err := row.Scan(
		&w.ID, &w.UserID, &w.Domain, &aliases, &w.Type, &w.Status,
		&w.DocumentRoot, &w.ProxyTarget, &w.ContainerID, &w.PHPVersion,
		&w.SSLEnabled, &w.SSLCertID, &w.IPV6, &w.AccessLog, &w.ErrorLog,
		&w.DefaultServer, &w.NginxCustom, &w.Remark, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	w.Alias = aliases
	return &w, nil
}

func scanWebsiteRows(rows pgx.Rows) ([]Website, error) {
	var websites []Website
	for rows.Next() {
		var w Website
		err := rows.Scan(
			&w.ID, &w.UserID, &w.Domain, &w.Alias, &w.Type, &w.Status,
			&w.DocumentRoot, &w.ProxyTarget, &w.ContainerID, &w.PHPVersion,
			&w.SSLEnabled, &w.SSLCertID, &w.IPV6, &w.AccessLog, &w.ErrorLog,
			&w.DefaultServer, &w.NginxCustom, &w.Remark, &w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		websites = append(websites, w)
	}
	return websites, nil
}

func (r *Repository) Create(ctx context.Context, w *Website) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO websites (user_id, domain, alias, type, status, document_root, proxy_target,
			php_version, ipv6, access_log, error_log, remark)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, created_at, updated_at`,
		w.UserID, w.Domain, w.Alias, w.Type, w.Status, w.DocumentRoot,
		w.ProxyTarget, w.PHPVersion, w.IPV6, w.AccessLog, w.ErrorLog, w.Remark,
	).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Website, error) {
	w, err := scanWebsite(r.pool.QueryRow(ctx,
		`SELECT `+websiteCols+` FROM websites WHERE id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return w, err
}

func (r *Repository) GetByDomain(ctx context.Context, domain string) (*Website, error) {
	w, err := scanWebsite(r.pool.QueryRow(ctx,
		`SELECT `+websiteCols+` FROM websites WHERE domain = $1`, domain))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return w, err
}

func (r *Repository) GetByAlias(ctx context.Context, alias string) (*Website, error) {
	w, err := scanWebsite(r.pool.QueryRow(ctx,
		`SELECT `+websiteCols+` FROM websites WHERE alias = $1`, alias))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return w, err
}

func (r *Repository) List(ctx context.Context, q ListQuery) ([]Website, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	idx := 1

	if q.Search != "" {
		where = append(where, fmt.Sprintf("(domain ILIKE $%d OR alias ILIKE $%d OR remark ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+q.Search+"%")
		idx++
	}
	if q.Type != "" {
		where = append(where, fmt.Sprintf("type = $%d", idx))
		args = append(args, q.Type)
		idx++
	}
	if q.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, q.Status)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM websites WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count websites: %w", err)
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 || q.Size > 100 {
		q.Size = 20
	}
	offset := (q.Page - 1) * q.Size

	args = append(args, q.Size, offset)
	query := fmt.Sprintf("SELECT %s FROM websites WHERE %s ORDER BY id DESC LIMIT $%d OFFSET $%d",
		websiteCols, whereClause, idx, idx+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list websites: %w", err)
	}
	defer rows.Close()

	websites, err := scanWebsiteRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return websites, total, nil
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) error {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	idx := 1

	if req.Domain != nil {
		sets = append(sets, fmt.Sprintf("domain = $%d", idx))
		args = append(args, *req.Domain)
		idx++
	}
	if req.ProxyTarget != nil {
		sets = append(sets, fmt.Sprintf("proxy_target = $%d", idx))
		args = append(args, *req.ProxyTarget)
		idx++
	}
	if req.PHPVersion != nil {
		sets = append(sets, fmt.Sprintf("php_version = $%d", idx))
		args = append(args, *req.PHPVersion)
		idx++
	}
	if req.IPV6 != nil {
		sets = append(sets, fmt.Sprintf("ipv6 = $%d", idx))
		args = append(args, *req.IPV6)
		idx++
	}
	if req.DefaultServer != nil {
		sets = append(sets, fmt.Sprintf("default_server = $%d", idx))
		args = append(args, *req.DefaultServer)
		idx++
	}
	if req.AccessLog != nil {
		sets = append(sets, fmt.Sprintf("access_log = $%d", idx))
		args = append(args, *req.AccessLog)
		idx++
	}
	if req.ErrorLog != nil {
		sets = append(sets, fmt.Sprintf("error_log = $%d", idx))
		args = append(args, *req.ErrorLog)
		idx++
	}
	if req.Remark != nil {
		sets = append(sets, fmt.Sprintf("remark = $%d", idx))
		args = append(args, *req.Remark)
		idx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE websites SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE websites SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	return err
}

func (r *Repository) UpdateNginxCustom(ctx context.Context, id int64, config string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE websites SET nginx_custom = $1, updated_at = NOW() WHERE id = $2", config, id)
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM websites WHERE id = $1", id)
	return err
}

// Domains

func (r *Repository) CreateDomain(ctx context.Context, d *WebsiteDomain) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO website_domains (website_id, domain, port, ssl)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		d.WebsiteID, d.Domain, d.Port, d.SSL,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *Repository) ListDomains(ctx context.Context, websiteID int64) ([]WebsiteDomain, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, website_id, domain, port, ssl, created_at
		 FROM website_domains WHERE website_id = $1 ORDER BY id`, websiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []WebsiteDomain
	for rows.Next() {
		var d WebsiteDomain
		if err := rows.Scan(&d.ID, &d.WebsiteID, &d.Domain, &d.Port, &d.SSL, &d.CreatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func (r *Repository) DeleteDomain(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM website_domains WHERE id = $1", id)
	return err
}

func (r *Repository) DomainExists(ctx context.Context, domain string, port int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM website_domains WHERE domain = $1 AND port = $2)",
		domain, port).Scan(&exists)
	return exists, err
}
