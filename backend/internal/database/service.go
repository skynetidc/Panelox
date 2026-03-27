package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrInvalidType = errors.New("invalid database type (use: mysql, postgresql)")
	ErrNameTaken   = errors.New("name already taken")
)

var defaultVersions = map[string]string{
	TypeMySQL:      "8.4",
	TypePostgreSQL: "16",
}

var defaultImages = map[string]string{
	TypeMySQL:      "mysql",
	TypePostgreSQL: "postgres",
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Server management

func (s *Service) CreateServer(ctx context.Context, userID int64, req CreateServerRequest) (*DBServer, error) {
	if req.Type != TypeMySQL && req.Type != TypePostgreSQL {
		return nil, ErrInvalidType
	}

	version := req.Version
	if version == "" {
		version = defaultVersions[req.Type]
	}

	port := req.Port
	if port == 0 {
		if req.Type == TypeMySQL {
			port = 3306
		} else {
			port = 5432
		}
	}

	containerName := fmt.Sprintf("panelox-db-%s", strings.ToLower(req.Name))

	srv := &DBServer{
		UserID:       userID,
		Name:         req.Name,
		Type:         req.Type,
		Version:      version,
		Host:         "127.0.0.1",
		Port:         port,
		RootPassword: req.RootPassword,
		ContainerID:  "",
		Status:       StatusCreating,
	}

	if err := s.repo.CreateServer(ctx, srv); err != nil {
		return nil, fmt.Errorf("create server record: %w", err)
	}

	// Spin up Docker container asynchronously
	go s.provisionContainer(srv.ID, containerName, req.Type, version, port, req.RootPassword)

	return srv, nil
}

func (s *Service) provisionContainer(serverID int64, name, dbType, version string, port int, rootPassword string) {
	ctx := context.Background()
	image := fmt.Sprintf("%s:%s-alpine", defaultImages[dbType], version)

	// Build docker run args
	args := []string{
		"run", "-d",
		"--name", name,
		"--restart", "unless-stopped",
		"-p", fmt.Sprintf("%d:%d", port, defaultPort(dbType)),
		"--label", "panelox.managed=true",
		"--label", fmt.Sprintf("panelox.type=%s", dbType),
	}

	// Add environment variables
	switch dbType {
	case TypeMySQL:
		args = append(args, "-e", fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", rootPassword))
	case TypePostgreSQL:
		args = append(args, "-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", rootPassword))
		image = fmt.Sprintf("postgres:%s-alpine", version)
	}

	// Add volume for data persistence
	volumeName := fmt.Sprintf("panelox-db-%s-data", name)
	switch dbType {
	case TypeMySQL:
		args = append(args, "-v", fmt.Sprintf("%s:/var/lib/mysql", volumeName))
	case TypePostgreSQL:
		args = append(args, "-v", fmt.Sprintf("%s:/var/lib/postgresql/data", volumeName))
	}

	args = append(args, image)

	// Run docker container
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to create container %s: %v, output: %s", name, err, string(output))
		s.repo.UpdateServerStatus(ctx, serverID, StatusError, "")
		return
	}

	containerID := strings.TrimSpace(string(output))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	s.repo.UpdateServerStatus(ctx, serverID, StatusRunning, containerID)
	log.Printf("Database container %s created: %s", name, containerID)
}

func (s *Service) GetServer(ctx context.Context, id int64) (*DBServer, error) {
	srv, err := s.repo.GetServer(ctx, id)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, ErrNotFound
	}
	return srv, nil
}

func (s *Service) ListServers(ctx context.Context, page, size int) ([]DBServer, int64, error) {
	return s.repo.ListServers(ctx, page, size)
}

func (s *Service) DeleteServer(ctx context.Context, id int64) error {
	srv, err := s.repo.GetServer(ctx, id)
	if err != nil {
		return err
	}
	if srv == nil {
		return ErrNotFound
	}

	// Stop and remove container
	if srv.ContainerID != "" {
		exec.Command("docker", "stop", srv.ContainerID).Run()
		exec.Command("docker", "rm", srv.ContainerID).Run()
	}

	return s.repo.DeleteServer(ctx, id)
}

func (s *Service) OperateServer(ctx context.Context, id int64, action string) error {
	srv, err := s.repo.GetServer(ctx, id)
	if err != nil {
		return err
	}
	if srv == nil {
		return ErrNotFound
	}
	if srv.ContainerID == "" {
		return fmt.Errorf("no container associated")
	}

	switch action {
	case "start":
		if out, err := exec.Command("docker", "start", srv.ContainerID).CombinedOutput(); err != nil {
			return fmt.Errorf("start failed: %s", string(out))
		}
		return s.repo.UpdateServerStatus(ctx, id, StatusRunning, srv.ContainerID)
	case "stop":
		if out, err := exec.Command("docker", "stop", srv.ContainerID).CombinedOutput(); err != nil {
			return fmt.Errorf("stop failed: %s", string(out))
		}
		return s.repo.UpdateServerStatus(ctx, id, StatusStopped, srv.ContainerID)
	case "restart":
		if out, err := exec.Command("docker", "restart", srv.ContainerID).CombinedOutput(); err != nil {
			return fmt.Errorf("restart failed: %s", string(out))
		}
		return s.repo.UpdateServerStatus(ctx, id, StatusRunning, srv.ContainerID)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// Database management

func (s *Service) CreateDatabase(ctx context.Context, serverID int64, req CreateDatabaseRequest) (*Database, error) {
	srv, err := s.repo.GetServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, ErrNotFound
	}

	charset := req.Charset
	if charset == "" {
		if srv.Type == TypeMySQL {
			charset = "utf8mb4"
		} else {
			charset = "UTF8"
		}
	}

	// Execute CREATE DATABASE via docker exec
	var cmdArgs []string
	switch srv.Type {
	case TypeMySQL:
		sql := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s;", req.Name, charset)
		cmdArgs = []string{"exec", srv.ContainerID, "mysql", "-uroot", fmt.Sprintf("-p%s", srv.RootPassword), "-e", sql}
	case TypePostgreSQL:
		sql := fmt.Sprintf("CREATE DATABASE \"%s\" ENCODING '%s';", req.Name, charset)
		cmdArgs = []string{"exec", srv.ContainerID, "psql", "-U", "postgres", "-c", sql}
	}

	if out, err := exec.Command("docker", cmdArgs...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("create database failed: %s", string(out))
	}

	db := &Database{
		ServerID: serverID,
		Name:     req.Name,
		Charset:  charset,
	}
	if err := s.repo.CreateDatabase(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

func (s *Service) ListDatabases(ctx context.Context, serverID int64) ([]Database, error) {
	return s.repo.ListDatabases(ctx, serverID)
}

func (s *Service) DeleteDatabase(ctx context.Context, id int64) error {
	db, err := s.repo.GetDatabase(ctx, id)
	if err != nil {
		return err
	}
	if db == nil {
		return ErrNotFound
	}

	srv, err := s.repo.GetServer(ctx, db.ServerID)
	if err != nil {
		return err
	}
	if srv != nil && srv.ContainerID != "" {
		var cmdArgs []string
		switch srv.Type {
		case TypeMySQL:
			cmdArgs = []string{"exec", srv.ContainerID, "mysql", "-uroot", fmt.Sprintf("-p%s", srv.RootPassword), "-e", fmt.Sprintf("DROP DATABASE `%s`;", db.Name)}
		case TypePostgreSQL:
			cmdArgs = []string{"exec", srv.ContainerID, "psql", "-U", "postgres", "-c", fmt.Sprintf("DROP DATABASE \"%s\";", db.Name)}
		}
		exec.Command("docker", cmdArgs...).Run()
	}

	return s.repo.DeleteDatabase(ctx, id)
}

// DB User management

func (s *Service) CreateDBUser(ctx context.Context, serverID int64, req CreateDBUserRequest) (*DBUser, error) {
	srv, err := s.repo.GetServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, ErrNotFound
	}

	// Create user via docker exec
	switch srv.Type {
	case TypeMySQL:
		sql := fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s';", req.Username, req.Password)
		for _, dbName := range req.Databases {
			sql += fmt.Sprintf(" GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", dbName, req.Username)
		}
		sql += " FLUSH PRIVILEGES;"
		cmdArgs := []string{"exec", srv.ContainerID, "mysql", "-uroot", fmt.Sprintf("-p%s", srv.RootPassword), "-e", sql}
		if out, err := exec.Command("docker", cmdArgs...).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("create user failed: %s", string(out))
		}
	case TypePostgreSQL:
		sql := fmt.Sprintf("CREATE USER \"%s\" WITH PASSWORD '%s';", req.Username, req.Password)
		for _, dbName := range req.Databases {
			sql += fmt.Sprintf(" GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", dbName, req.Username)
		}
		cmdArgs := []string{"exec", srv.ContainerID, "psql", "-U", "postgres", "-c", sql}
		if out, err := exec.Command("docker", cmdArgs...).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("create user failed: %s", string(out))
		}
	}

	u := &DBUser{
		ServerID:  serverID,
		Username:  req.Username,
		Password:  req.Password,
		Databases: req.Databases,
	}
	if err := s.repo.CreateDBUser(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) ListDBUsers(ctx context.Context, serverID int64) ([]DBUser, error) {
	return s.repo.ListDBUsers(ctx, serverID)
}

func (s *Service) DeleteDBUser(ctx context.Context, id int64) error {
	return s.repo.DeleteDBUser(ctx, id)
}

func defaultPort(dbType string) int {
	if dbType == TypeMySQL {
		return 3306
	}
	return 5432
}
