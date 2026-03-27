package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// DB Servers

func (r *Repository) CreateServer(ctx context.Context, s *DBServer) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO db_servers (user_id, name, type, version, host, port, root_password, container_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		s.UserID, s.Name, s.Type, s.Version, s.Host, s.Port, s.RootPassword, s.ContainerID, s.Status,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *Repository) GetServer(ctx context.Context, id int64) (*DBServer, error) {
	var s DBServer
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, type, version, host, port, root_password, container_id, status, created_at, updated_at
		 FROM db_servers WHERE id = $1`, id,
	).Scan(&s.ID, &s.UserID, &s.Name, &s.Type, &s.Version, &s.Host, &s.Port, &s.RootPassword, &s.ContainerID, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListServers(ctx context.Context, page, size int) ([]DBServer, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM db_servers").Scan(&total); err != nil {
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
		`SELECT id, user_id, name, type, version, host, port, '', container_id, status, created_at, updated_at
		 FROM db_servers ORDER BY id DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var servers []DBServer
	for rows.Next() {
		var s DBServer
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Type, &s.Version, &s.Host, &s.Port, &s.RootPassword, &s.ContainerID, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		servers = append(servers, s)
	}
	return servers, total, nil
}

func (r *Repository) UpdateServerStatus(ctx context.Context, id int64, status, containerID string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE db_servers SET status = $1, container_id = $2, updated_at = NOW() WHERE id = $3",
		status, containerID, id)
	return err
}

func (r *Repository) DeleteServer(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM db_servers WHERE id = $1", id)
	return err
}

// Databases

func (r *Repository) CreateDatabase(ctx context.Context, d *Database) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO databases (server_id, name, charset) VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		d.ServerID, d.Name, d.Charset,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *Repository) ListDatabases(ctx context.Context, serverID int64) ([]Database, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, server_id, name, charset, created_at FROM databases WHERE server_id = $1 ORDER BY id`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []Database
	for rows.Next() {
		var d Database
		if err := rows.Scan(&d.ID, &d.ServerID, &d.Name, &d.Charset, &d.CreatedAt); err != nil {
			return nil, err
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

func (r *Repository) DeleteDatabase(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM databases WHERE id = $1", id)
	return err
}

func (r *Repository) GetDatabase(ctx context.Context, id int64) (*Database, error) {
	var d Database
	err := r.pool.QueryRow(ctx,
		`SELECT id, server_id, name, charset, created_at FROM databases WHERE id = $1`, id,
	).Scan(&d.ID, &d.ServerID, &d.Name, &d.Charset, &d.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// DB Users

func (r *Repository) CreateDBUser(ctx context.Context, u *DBUser) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO db_users (server_id, username, password, privileges) VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		u.ServerID, u.Username, u.Password, "{}",
	).Scan(&u.ID, &u.CreatedAt)
}

func (r *Repository) ListDBUsers(ctx context.Context, serverID int64) ([]DBUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, server_id, username, '', created_at FROM db_users WHERE server_id = $1 ORDER BY id`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []DBUser
	for rows.Next() {
		var u DBUser
		if err := rows.Scan(&u.ID, &u.ServerID, &u.Username, &u.Password, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *Repository) DeleteDBUser(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM db_users WHERE id = $1", id)
	return err
}
