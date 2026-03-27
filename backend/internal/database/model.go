package database

import "time"

// Server types
const (
	TypeMySQL      = "mysql"
	TypePostgreSQL = "postgresql"
)

// Server statuses
const (
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusCreating = "creating"
	StatusError    = "error"
)

type DBServer struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Version      string    `json:"version"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	RootPassword string    `json:"-"`
	ContainerID  string    `json:"container_id,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Database struct {
	ID        int64     `json:"id"`
	ServerID  int64     `json:"server_id"`
	Name      string    `json:"name"`
	Charset   string    `json:"charset"`
	CreatedAt time.Time `json:"created_at"`
}

type DBUser struct {
	ID        int64     `json:"id"`
	ServerID  int64     `json:"server_id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Databases []string  `json:"databases,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Request DTOs

type CreateServerRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	RootPassword string `json:"root_password"`
	Port         int    `json:"port"`
}

type CreateDatabaseRequest struct {
	Name    string `json:"name"`
	Charset string `json:"charset"`
}

type CreateDBUserRequest struct {
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Databases []string `json:"databases"`
}
