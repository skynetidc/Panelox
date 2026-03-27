package ssl

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	panelacme "github.com/jakub/panelox/internal/pkg/acme"
	"github.com/jakub/panelox/internal/pkg/nginx"
)

var (
	ErrNotFound       = errors.New("certificate not found")
	ErrAccountNotFound = errors.New("ACME account not found")
	ErrCertInUse      = errors.New("certificate is in use by websites")
	ErrInvalidPEM     = errors.New("invalid PEM data")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ACME Account management

func (s *Service) CreateAcmeAccount(ctx context.Context, req CreateAcmeAccountRequest) (*AcmeAccount, error) {
	if req.Provider == "" {
		req.Provider = ProviderLetsEncrypt
	}
	if req.KeyType == "" {
		req.KeyType = "EC256"
	}

	// Generate account key
	key, keyPEM, err := panelacme.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	// Determine directory URL
	dirURL := panelacme.LetsEncryptURL
	if req.Provider == "letsencrypt-staging" {
		dirURL = panelacme.LetsEncryptStagingURL
		req.Provider = "letsencrypt"
	}

	// Register with ACME server
	client := panelacme.NewClient(key, dirURL)
	acct, err := client.GetOrRegister(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("register ACME account: %w", err)
	}

	account := &AcmeAccount{
		Email:      req.Email,
		Provider:   req.Provider,
		PrivateKey: keyPEM,
		URL:        acct.URI,
		KeyType:    req.KeyType,
	}

	if err := s.repo.CreateAcmeAccount(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}

func (s *Service) ListAcmeAccounts(ctx context.Context) ([]AcmeAccount, error) {
	return s.repo.ListAcmeAccounts(ctx)
}

func (s *Service) DeleteAcmeAccount(ctx context.Context, id int64) error {
	return s.repo.DeleteAcmeAccount(ctx, id)
}

// Certificate management

func (s *Service) Issue(ctx context.Context, userID int64, req IssueRequest) (*Certificate, error) {
	// Get ACME account
	account, err := s.repo.GetAcmeAccount(ctx, req.AcmeAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}

	if req.HTTPConfig == "" {
		req.HTTPConfig = HTTPToHTTPS
	}

	// Create certificate record in pending state
	cert := &Certificate{
		UserID:        userID,
		AcmeAccountID: &req.AcmeAccountID,
		Domains:       req.Domains,
		Provider:      ProviderLetsEncrypt,
		KeyType:       req.KeyType,
		AutoRenew:     req.AutoRenew,
		Status:        StatusIssuing,
		HTTPConfig:    req.HTTPConfig,
		Description:   req.Description,
	}

	if err := s.repo.CreateCertificate(ctx, cert); err != nil {
		return nil, fmt.Errorf("create certificate record: %w", err)
	}

	// Issue asynchronously
	go s.issueCertificate(cert.ID, account, req.Domains)

	return cert, nil
}

func (s *Service) issueCertificate(certID int64, account *AcmeAccount, domains []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Parse account key
	key, err := panelacme.ParseKey(account.PrivateKey)
	if err != nil {
		s.repo.UpdateCertificateStatus(ctx, certID, StatusFailed, fmt.Sprintf("parse account key: %v", err))
		return
	}

	client := panelacme.NewClient(key, panelacme.LetsEncryptURL)
	if _, err := client.GetOrRegister(ctx, account.Email); err != nil {
		s.repo.UpdateCertificateStatus(ctx, certID, StatusFailed, fmt.Sprintf("register: %v", err))
		return
	}

	// HTTP-01 challenge: write token files to well-known path
	challengeDir := "/www/acme-challenge"
	os.MkdirAll(challengeDir, 0755)

	challengeHandler := func(domain, token, keyAuth string) error {
		path := filepath.Join(challengeDir, token)
		return os.WriteFile(path, []byte(keyAuth), 0644)
	}

	cleanupHandler := func(domain, token string) {
		os.Remove(filepath.Join(challengeDir, token))
	}

	result, err := client.ObtainCertificateHTTP01(ctx, domains, challengeHandler, cleanupHandler)
	if err != nil {
		s.repo.UpdateCertificateStatus(ctx, certID, StatusFailed, err.Error())
		log.Printf("SSL issuance failed for cert %d: %v", certID, err)
		return
	}

	// Save certificate data
	if err := s.repo.UpdateCertificateData(ctx, certID,
		result.CertPEM, result.KeyPEM, result.ChainPEM,
		result.ExpiresAt, result.IssuedAt, result.CertURL); err != nil {
		log.Printf("Failed to save certificate data for cert %d: %v", certID, err)
		return
	}

	log.Printf("SSL certificate issued for cert %d, domains: %v, expires: %s", certID, domains, result.ExpiresAt)
}

func (s *Service) Upload(ctx context.Context, userID int64, req UploadRequest) (*Certificate, error) {
	// Parse certificate PEM
	block, _ := pem.Decode([]byte(req.Certificate))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, ErrInvalidPEM
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	// Validate private key
	keyBlock, _ := pem.Decode([]byte(req.PrivateKey))
	if keyBlock == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}

	// Extract domains
	domains := parsedCert.DNSNames
	if len(domains) == 0 && parsedCert.Subject.CommonName != "" {
		domains = []string{parsedCert.Subject.CommonName}
	}

	cert := &Certificate{
		UserID:      userID,
		Domains:     domains,
		Provider:    ProviderCustom,
		KeyType:     "custom",
		CertPEM:     req.Certificate,
		KeyPEM:      req.PrivateKey,
		AutoRenew:   false,
		ExpiresAt:   &parsedCert.NotAfter,
		IssuedAt:    &parsedCert.NotBefore,
		Status:      StatusIssued,
		HTTPConfig:  HTTPToHTTPS,
		Description: req.Description,
	}

	if err := s.repo.CreateCertificate(ctx, cert); err != nil {
		return nil, err
	}

	return cert, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Certificate, error) {
	cert, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return nil, err
	}
	if cert == nil {
		return nil, ErrNotFound
	}
	return cert, nil
}

func (s *Service) List(ctx context.Context, page, size int) ([]Certificate, int64, error) {
	return s.repo.ListCertificates(ctx, page, size)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	inUse, err := s.repo.CertificateInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrCertInUse
	}

	return s.repo.DeleteCertificate(ctx, id)
}

func (s *Service) Renew(ctx context.Context, id int64) error {
	cert, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return err
	}
	if cert == nil {
		return ErrNotFound
	}
	if cert.Provider != ProviderLetsEncrypt || cert.AcmeAccountID == nil {
		return fmt.Errorf("only Let's Encrypt certificates can be auto-renewed")
	}

	account, err := s.repo.GetAcmeAccount(ctx, *cert.AcmeAccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return ErrAccountNotFound
	}

	s.repo.UpdateCertificateStatus(ctx, id, StatusIssuing, "renewal in progress")
	go s.issueCertificate(id, account, cert.Domains)

	return nil
}

// Apply applies an SSL certificate to a website's nginx config.
func (s *Service) Apply(ctx context.Context, req ApplyRequest) error {
	cert, err := s.repo.GetCertificate(ctx, req.CertificateID)
	if err != nil {
		return err
	}
	if cert == nil {
		return ErrNotFound
	}
	if cert.Status != StatusIssued {
		return fmt.Errorf("certificate not yet issued (status: %s)", cert.Status)
	}

	// Get website alias
	alias, err := s.repo.GetWebsiteAlias(ctx, req.WebsiteID)
	if err != nil {
		return fmt.Errorf("get website: %w", err)
	}

	// Write PEM files to site SSL directory
	paths := nginx.GetSitePaths(alias)
	os.MkdirAll(paths.SSL, 0755)

	if err := os.WriteFile(filepath.Join(paths.SSL, "fullchain.pem"), []byte(cert.CertPEM), 0644); err != nil {
		return fmt.Errorf("write fullchain: %w", err)
	}
	if err := os.WriteFile(filepath.Join(paths.SSL, "privkey.pem"), []byte(cert.KeyPEM), 0600); err != nil {
		return fmt.Errorf("write privkey: %w", err)
	}

	// Update website record
	if err := s.repo.ApplyToWebsite(ctx, req.WebsiteID, req.CertificateID); err != nil {
		return fmt.Errorf("apply to website: %w", err)
	}

	// Reload nginx
	nginx.TestAndReload()

	return nil
}

// RemoveFromWebsite removes SSL from a website.
func (s *Service) RemoveFromWebsite(ctx context.Context, websiteID int64) error {
	if err := s.repo.RemoveFromWebsite(ctx, websiteID); err != nil {
		return err
	}

	nginx.TestAndReload()
	return nil
}

// RenewExpiring checks and renews certificates expiring within 30 days.
// Called by a background cron job.
func (s *Service) RenewExpiring(ctx context.Context) {
	certs, err := s.repo.GetAutoRenewCertificates(ctx)
	if err != nil {
		log.Printf("Failed to get auto-renew certificates: %v", err)
		return
	}

	for _, cert := range certs {
		log.Printf("Auto-renewing certificate %d for domains: %v", cert.ID, cert.Domains)
		if err := s.Renew(ctx, cert.ID); err != nil {
			log.Printf("Failed to renew certificate %d: %v", cert.ID, err)
		}
	}
}
