package website

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jakub/panelox/internal/pkg/nginx"
)

var (
	ErrDomainTaken   = errors.New("domain already in use")
	ErrAliasTaken    = errors.New("alias already in use")
	ErrNotFound      = errors.New("website not found")
	ErrInvalidType   = errors.New("invalid website type")
	ErrInvalidDomain = errors.New("invalid domain name")

	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	aliasRegex  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID int64, req CreateRequest) (*Website, error) {
	// Validate type
	switch req.Type {
	case TypeStatic, TypeProxy, TypePHP, TypeNode:
	default:
		return nil, ErrInvalidType
	}

	// Validate domain
	if !domainRegex.MatchString(req.Domain) {
		return nil, ErrInvalidDomain
	}

	// Generate alias from domain if not provided
	alias := req.Alias
	if alias == "" {
		alias = generateAlias(req.Domain)
	}
	if !aliasRegex.MatchString(alias) {
		return nil, fmt.Errorf("invalid alias: must be lowercase alphanumeric with hyphens")
	}

	// Check uniqueness
	existing, err := s.repo.GetByDomain(ctx, req.Domain)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDomainTaken
	}

	existingAlias, err := s.repo.GetByAlias(ctx, alias)
	if err != nil {
		return nil, err
	}
	if existingAlias != nil {
		return nil, ErrAliasTaken
	}

	paths := nginx.GetSitePaths(alias)

	w := &Website{
		UserID:       userID,
		Domain:       req.Domain,
		Alias:        alias,
		Type:         req.Type,
		Status:       StatusActive,
		DocumentRoot: paths.Index,
		ProxyTarget:  req.ProxyTarget,
		PHPVersion:   req.PHPVersion,
		IPV6:         req.IPV6,
		AccessLog:    true,
		ErrorLog:     true,
		Remark:       req.Remark,
	}

	// Create website record
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create website: %w", err)
	}

	// Create primary domain entry
	primaryDomain := &WebsiteDomain{
		WebsiteID: w.ID,
		Domain:    req.Domain,
		Port:      80,
	}
	if err := s.repo.CreateDomain(ctx, primaryDomain); err != nil {
		return nil, fmt.Errorf("create primary domain: %w", err)
	}

	// Create additional domains
	for _, d := range req.Domains {
		port := d.Port
		if port == 0 {
			port = 80
		}
		dom := &WebsiteDomain{
			WebsiteID: w.ID,
			Domain:    d.Domain,
			Port:      port,
		}
		if err := s.repo.CreateDomain(ctx, dom); err != nil {
			return nil, fmt.Errorf("create domain %s: %w", d.Domain, err)
		}
	}

	// Create site directories
	for _, dir := range paths.AllDirs() {
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Log but don't fail - directories may not be creatable on dev machine
			fmt.Printf("Warning: could not create directory %s: %v\n", dir, err)
		}
	}

	// Create default index.html
	indexPath := paths.Index + "/index.html"
	if err := os.WriteFile(indexPath, []byte(nginx.GenerateDefaultIndex(req.Domain)), 0644); err != nil {
		fmt.Printf("Warning: could not create index.html: %v\n", err)
	}

	// Generate and write nginx config
	if err := s.generateNginxConfig(ctx, w); err != nil {
		fmt.Printf("Warning: could not generate nginx config: %v\n", err)
	}

	// Load domains
	w.Domains, _ = s.repo.ListDomains(ctx, w.ID)

	return w, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Website, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrNotFound
	}

	w.Domains, _ = s.repo.ListDomains(ctx, w.ID)
	return w, nil
}

func (s *Service) List(ctx context.Context, q ListQuery) ([]Website, int64, error) {
	return s.repo.List(ctx, q)
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) error {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrNotFound
	}

	if err := s.repo.Update(ctx, id, req); err != nil {
		return err
	}

	// Regenerate nginx config
	updated, _ := s.repo.GetByID(ctx, id)
	if updated != nil {
		s.generateNginxConfig(ctx, updated)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrNotFound
	}

	// Delete from database
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Remove nginx config
	paths := nginx.GetSitePaths(w.Alias)
	os.Remove(paths.ConfFile)

	// Remove site directory
	os.RemoveAll(paths.Root)

	// Reload nginx
	nginx.TestAndReload()

	return nil
}

func (s *Service) Operate(ctx context.Context, id int64, action string) error {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrNotFound
	}

	switch action {
	case "start":
		if err := s.repo.UpdateStatus(ctx, id, StatusActive); err != nil {
			return err
		}
		w.Status = StatusActive
		return s.generateNginxConfig(ctx, w)

	case "stop":
		if err := s.repo.UpdateStatus(ctx, id, StatusStopped); err != nil {
			return err
		}
		// Remove nginx config to stop serving
		paths := nginx.GetSitePaths(w.Alias)
		os.Remove(paths.ConfFile)
		nginx.TestAndReload()
		return nil

	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (s *Service) GetNginxConfig(ctx context.Context, id int64) (string, error) {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if w == nil {
		return "", ErrNotFound
	}

	paths := nginx.GetSitePaths(w.Alias)
	content, err := os.ReadFile(paths.ConfFile)
	if err != nil {
		// Generate from template if file doesn't exist
		return s.buildNginxConfig(ctx, w)
	}
	return string(content), nil
}

func (s *Service) UpdateNginxConfig(ctx context.Context, id int64, config string) error {
	w, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrNotFound
	}

	paths := nginx.GetSitePaths(w.Alias)
	if err := os.WriteFile(paths.ConfFile, []byte(config), 0644); err != nil {
		return fmt.Errorf("write nginx config: %w", err)
	}

	// Test and reload
	if err := nginx.TestAndReload(); err != nil {
		// Rollback: regenerate from template
		s.generateNginxConfig(ctx, w)
		return fmt.Errorf("nginx config invalid: %w", err)
	}

	// Save custom config to DB
	s.repo.UpdateNginxCustom(ctx, id, config)
	return nil
}

// Domain management

func (s *Service) AddDomain(ctx context.Context, websiteID int64, req DomainCreateRequest) (*WebsiteDomain, error) {
	w, err := s.repo.GetByID(ctx, websiteID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrNotFound
	}

	port := req.Port
	if port == 0 {
		port = 80
	}

	exists, err := s.repo.DomainExists(ctx, req.Domain, port)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDomainTaken
	}

	d := &WebsiteDomain{
		WebsiteID: websiteID,
		Domain:    req.Domain,
		Port:      port,
	}
	if err := s.repo.CreateDomain(ctx, d); err != nil {
		return nil, err
	}

	// Regenerate nginx config
	s.generateNginxConfig(ctx, w)
	return d, nil
}

func (s *Service) ListDomains(ctx context.Context, websiteID int64) ([]WebsiteDomain, error) {
	return s.repo.ListDomains(ctx, websiteID)
}

func (s *Service) DeleteDomain(ctx context.Context, websiteID, domainID int64) error {
	w, err := s.repo.GetByID(ctx, websiteID)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrNotFound
	}

	if err := s.repo.DeleteDomain(ctx, domainID); err != nil {
		return err
	}

	// Regenerate nginx config
	s.generateNginxConfig(ctx, w)
	return nil
}

// Internal helpers

func (s *Service) generateNginxConfig(ctx context.Context, w *Website) error {
	config, err := s.buildNginxConfig(ctx, w)
	if err != nil {
		return err
	}

	paths := nginx.GetSitePaths(w.Alias)

	// Ensure conf.d directory exists
	os.MkdirAll("/www/conf.d", 0755)

	if err := os.WriteFile(paths.ConfFile, []byte(config), 0644); err != nil {
		return fmt.Errorf("write nginx config: %w", err)
	}

	// Best effort reload
	nginx.TestAndReload()
	return nil
}

func (s *Service) buildNginxConfig(ctx context.Context, w *Website) (string, error) {
	domains, err := s.repo.ListDomains(ctx, w.ID)
	if err != nil {
		return "", err
	}

	nginxDomains := make([]nginx.Domain, len(domains))
	for i, d := range domains {
		nginxDomains[i] = nginx.Domain{
			Name: d.Domain,
			Port: d.Port,
			SSL:  d.SSL,
		}
	}

	// Map PHP version to port
	phpPort := phpVersionToPort(w.PHPVersion)

	cfg := nginx.Config{
		Alias:         w.Alias,
		Domains:       nginxDomains,
		Type:          w.Type,
		DocumentRoot:  w.DocumentRoot,
		ProxyTarget:   w.ProxyTarget,
		PHPVersion:    w.PHPVersion,
		PHPPort:       phpPort,
		IPV6:          w.IPV6,
		AccessLog:     w.AccessLog,
		ErrorLog:      w.ErrorLog,
		DefaultServer: w.DefaultServer,
		SSLEnabled:    w.SSLEnabled,
		CustomConfig:  w.NginxCustom,
	}

	return nginx.GenerateServerBlock(cfg), nil
}

func generateAlias(domain string) string {
	alias := strings.ReplaceAll(domain, ".", "-")
	alias = strings.ToLower(alias)
	return alias
}

func phpVersionToPort(version string) int {
	ports := map[string]int{
		"8.3": 9000,
		"8.2": 9001,
		"8.1": 9002,
		"8.0": 9003,
		"7.4": 9004,
	}
	if p, ok := ports[version]; ok {
		return p
	}
	return 9000
}
