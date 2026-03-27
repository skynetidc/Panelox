package auth

import (
	"context"
	"fmt"
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

func (r *Repository) CreateUser(ctx context.Context, email, password, name, role string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password, name, role) VALUES ($1, $2, $3, $4)
		 RETURNING id, email, password, name, role, parent_id, status, language, created_at, updated_at`,
		email, password, name, role,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Role, &u.ParentID, &u.Status, &u.Language, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password, name, role, parent_id, status, language, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Role, &u.ParentID, &u.Status, &u.Language, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password, name, role, parent_id, status, language, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Role, &u.ParentID, &u.Status, &u.Language, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *Repository) ListUsers(ctx context.Context, page, size int) ([]User, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, password, name, role, parent_id, status, language, created_at, updated_at
		 FROM users ORDER BY id LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Password, &u.Name, &u.Role, &u.ParentID, &u.Status, &u.Language, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (r *Repository) UpdateUser(ctx context.Context, id int64, req UpdateUserRequest) error {
	query := "UPDATE users SET updated_at = NOW()"
	args := []any{}
	argIdx := 1

	if req.Email != nil {
		query += fmt.Sprintf(", email = $%d", argIdx)
		args = append(args, *req.Email)
		argIdx++
	}
	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Password != nil {
		query += fmt.Sprintf(", password = $%d", argIdx)
		args = append(args, *req.Password)
		argIdx++
	}
	if req.Role != nil {
		query += fmt.Sprintf(", role = $%d", argIdx)
		args = append(args, *req.Role)
		argIdx++
	}
	if req.Status != nil {
		query += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, *req.Status)
		argIdx++
	}
	if req.Language != nil {
		query += fmt.Sprintf(", language = $%d", argIdx)
		args = append(args, *req.Language)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

func (r *Repository) UserCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// Sessions

func (r *Repository) CreateSession(ctx context.Context, userID int64, refreshToken, userAgent, ip string) (*Session, error) {
	var s Session
	err := r.pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id, refresh_token, user_agent, ip_address, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at`,
		userID, refreshToken, userAgent, ip, time.Now().Add(refreshTokenDuration),
	).Scan(&s.ID, &s.UserID, &s.RefreshToken, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetSessionByToken(ctx context.Context, token string) (*Session, error) {
	var s Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token, user_agent, ip_address, expires_at, created_at
		 FROM sessions WHERE refresh_token = $1 AND expires_at > NOW()`, token,
	).Scan(&s.ID, &s.UserID, &s.RefreshToken, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &s, nil
}

func (r *Repository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM sessions WHERE refresh_token = $1", token)
	return err
}

func (r *Repository) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}
