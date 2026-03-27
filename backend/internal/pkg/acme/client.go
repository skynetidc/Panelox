package acme

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"golang.org/x/crypto/acme"
)

const (
	LetsEncryptURL        = "https://acme-v02.api.letsencrypt.org/directory"
	LetsEncryptStagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

// Client wraps the ACME client for certificate operations.
type Client struct {
	acmeClient *acme.Client
	account    *acme.Account
}

// GenerateKey creates a new ECDSA P-256 private key and returns it PEM-encoded.
func GenerateKey() (crypto.Signer, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("generate key: %w", err)
	}

	derBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, "", fmt.Errorf("marshal key: %w", err)
	}

	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derBytes,
	})

	return key, string(pemBlock), nil
}

// ParseKey parses a PEM-encoded private key.
func ParseKey(pemData string) (crypto.Signer, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	switch block.Type {
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("key does not implement crypto.Signer")
		}
		return signer, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}
}

// NewClient creates a new ACME client with an existing account key.
func NewClient(accountKey crypto.Signer, directoryURL string) *Client {
	return &Client{
		acmeClient: &acme.Client{
			Key:          accountKey,
			DirectoryURL: directoryURL,
		},
	}
}

// Register registers a new account with the ACME server.
func (c *Client) Register(ctx context.Context, email string) (*acme.Account, error) {
	account := &acme.Account{
		Contact: []string{"mailto:" + email},
	}

	acct, err := c.acmeClient.Register(ctx, account, acme.AcceptTOS)
	if err != nil {
		return nil, fmt.Errorf("register account: %w", err)
	}

	c.account = acct
	return acct, nil
}

// GetOrRegister gets an existing account or registers a new one.
func (c *Client) GetOrRegister(ctx context.Context, email string) (*acme.Account, error) {
	acct, err := c.acmeClient.GetReg(ctx, "")
	if err == nil {
		c.account = acct
		return acct, nil
	}

	return c.Register(ctx, email)
}

// CertificateResult holds the result of a certificate issuance.
type CertificateResult struct {
	CertPEM    string
	KeyPEM     string
	ChainPEM   string
	ExpiresAt  time.Time
	IssuedAt   time.Time
	CertURL    string
}

// ObtainCertificateHTTP01 obtains a certificate using HTTP-01 challenge.
// The challengeHandler is called with the token and key authorization that
// need to be served at /.well-known/acme-challenge/{token}.
func (c *Client) ObtainCertificateHTTP01(
	ctx context.Context,
	domains []string,
	challengeHandler func(domain, token, keyAuth string) error,
	cleanupHandler func(domain, token string),
) (*CertificateResult, error) {
	// Generate certificate key
	certKey, certKeyPEM, err := GenerateKey()
	if err != nil {
		return nil, err
	}

	// Create order
	order, err := c.acmeClient.AuthorizeOrder(ctx, acme.DomainIDs(domains...))
	if err != nil {
		return nil, fmt.Errorf("authorize order: %w", err)
	}

	// Handle authorizations
	for _, authURL := range order.AuthzURLs {
		auth, err := c.acmeClient.GetAuthorization(ctx, authURL)
		if err != nil {
			return nil, fmt.Errorf("get authorization: %w", err)
		}

		if auth.Status == acme.StatusValid {
			continue
		}

		// Find HTTP-01 challenge
		var challenge *acme.Challenge
		for _, ch := range auth.Challenges {
			if ch.Type == "http-01" {
				challenge = ch
				break
			}
		}
		if challenge == nil {
			return nil, fmt.Errorf("no HTTP-01 challenge found for %s", auth.Identifier.Value)
		}

		// Get key authorization
		keyAuth, err := c.acmeClient.HTTP01ChallengeResponse(challenge.Token)
		if err != nil {
			return nil, fmt.Errorf("challenge response: %w", err)
		}

		// Set up challenge response
		if err := challengeHandler(auth.Identifier.Value, challenge.Token, keyAuth); err != nil {
			return nil, fmt.Errorf("set challenge for %s: %w", auth.Identifier.Value, err)
		}
		defer cleanupHandler(auth.Identifier.Value, challenge.Token)

		// Accept challenge
		if _, err := c.acmeClient.Accept(ctx, challenge); err != nil {
			return nil, fmt.Errorf("accept challenge for %s: %w", auth.Identifier.Value, err)
		}

		// Wait for authorization
		if _, err := c.acmeClient.WaitAuthorization(ctx, authURL); err != nil {
			return nil, fmt.Errorf("wait authorization for %s: %w", auth.Identifier.Value, err)
		}
	}

	// Create CSR
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		DNSNames: domains,
	}, certKey)
	if err != nil {
		return nil, fmt.Errorf("create CSR: %w", err)
	}

	// Finalize order
	derChain, certURL, err := c.acmeClient.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	// Parse certificate for metadata
	cert, err := x509.ParseCertificate(derChain[0])
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	// Encode certificate chain to PEM
	var certPEM, chainPEM string
	for i, der := range derChain {
		pemBlock := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		})
		if i == 0 {
			certPEM = string(pemBlock)
		} else {
			chainPEM += string(pemBlock)
		}
	}

	// Build fullchain (cert + chain)
	fullchain := certPEM + chainPEM

	return &CertificateResult{
		CertPEM:   fullchain,
		KeyPEM:    certKeyPEM,
		ChainPEM:  chainPEM,
		ExpiresAt: cert.NotAfter,
		IssuedAt:  cert.NotBefore,
		CertURL:   certURL,
	}, nil
}
