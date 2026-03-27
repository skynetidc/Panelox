package dns

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

// Zones

func (r *Repository) CreateZone(ctx context.Context, z *Zone) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO dns_zones (user_id, name, status, serial)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		z.UserID, z.Name, z.Status, z.Serial,
	).Scan(&z.ID, &z.CreatedAt, &z.UpdatedAt)
}

func (r *Repository) GetZone(ctx context.Context, id int64) (*Zone, error) {
	var z Zone
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, status, serial, created_at, updated_at
		 FROM dns_zones WHERE id = $1`, id,
	).Scan(&z.ID, &z.UserID, &z.Name, &z.Status, &z.Serial, &z.CreatedAt, &z.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &z, err
}

func (r *Repository) GetZoneByName(ctx context.Context, name string) (*Zone, error) {
	var z Zone
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, status, serial, created_at, updated_at
		 FROM dns_zones WHERE name = $1`, name,
	).Scan(&z.ID, &z.UserID, &z.Name, &z.Status, &z.Serial, &z.CreatedAt, &z.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &z, err
}

func (r *Repository) ListZones(ctx context.Context, page, size int) ([]Zone, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM dns_zones").Scan(&total); err != nil {
		return nil, 0, err
	}
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }
	offset := (page - 1) * size

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, status, serial, created_at, updated_at
		 FROM dns_zones ORDER BY id DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var zones []Zone
	for rows.Next() {
		var z Zone
		if err := rows.Scan(&z.ID, &z.UserID, &z.Name, &z.Status, &z.Serial, &z.CreatedAt, &z.UpdatedAt); err != nil {
			return nil, 0, err
		}
		zones = append(zones, z)
	}
	return zones, total, nil
}

func (r *Repository) IncrementSerial(ctx context.Context, zoneID int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE dns_zones SET serial = serial + 1, updated_at = NOW() WHERE id = $1", zoneID)
	return err
}

func (r *Repository) DeleteZone(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM dns_zones WHERE id = $1", id)
	return err
}

// Records

func (r *Repository) CreateRecord(ctx context.Context, rec *Record) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO dns_records (zone_id, name, type, content, ttl, priority)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		rec.ZoneID, rec.Name, rec.Type, rec.Content, rec.TTL, rec.Priority,
	).Scan(&rec.ID, &rec.CreatedAt, &rec.UpdatedAt)
}

func (r *Repository) GetRecord(ctx context.Context, id int64) (*Record, error) {
	var rec Record
	err := r.pool.QueryRow(ctx,
		`SELECT id, zone_id, name, type, content, ttl, priority, created_at, updated_at
		 FROM dns_records WHERE id = $1`, id,
	).Scan(&rec.ID, &rec.ZoneID, &rec.Name, &rec.Type, &rec.Content, &rec.TTL, &rec.Priority, &rec.CreatedAt, &rec.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &rec, err
}

func (r *Repository) ListRecords(ctx context.Context, zoneID int64) ([]Record, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, zone_id, name, type, content, ttl, priority, created_at, updated_at
		 FROM dns_records WHERE zone_id = $1 ORDER BY type, name`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.ZoneID, &rec.Name, &rec.Type, &rec.Content, &rec.TTL, &rec.Priority, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *Repository) UpdateRecord(ctx context.Context, id int64, req UpdateRecordRequest) error {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	idx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", idx)); args = append(args, *req.Name); idx++
	}
	if req.Type != nil {
		sets = append(sets, fmt.Sprintf("type = $%d", idx)); args = append(args, *req.Type); idx++
	}
	if req.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", idx)); args = append(args, *req.Content); idx++
	}
	if req.TTL != nil {
		sets = append(sets, fmt.Sprintf("ttl = $%d", idx)); args = append(args, *req.TTL); idx++
	}
	if req.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", idx)); args = append(args, *req.Priority); idx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE dns_records SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *Repository) DeleteRecord(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM dns_records WHERE id = $1", id)
	return err
}

func (r *Repository) RecordCount(ctx context.Context, zoneID int64) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM dns_records WHERE zone_id = $1", zoneID).Scan(&count)
	return count, err
}
