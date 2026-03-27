package acme

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/acme"
)

// DNSProviderConfig holds credentials for a DNS provider.
type DNSProviderConfig struct {
	Provider string            `json:"provider"` // cloudflare, route53, digitalocean, etc.
	Credentials map[string]string `json:"credentials"`
}

// Supported DNS providers and their required credential fields.
var ProviderFields = map[string][]string{
	"cloudflare":   {"api_token"},
	"route53":      {"access_key_id", "secret_access_key", "region"},
	"digitalocean": {"api_token"},
	"godaddy":      {"api_key", "api_secret"},
	"namecheap":    {"api_user", "api_key", "client_ip"},
	"hetzner":      {"api_token"},
	"vultr":        {"api_key"},
	"linode":       {"api_token"},
	"ovh":          {"endpoint", "application_key", "application_secret", "consumer_key"},
	"porkbun":      {"api_key", "secret_key"},
}

// ObtainCertificateDNS01 obtains a certificate using DNS-01 challenge.
// The createRecord/deleteRecord functions handle DNS TXT record management.
func (c *Client) ObtainCertificateDNS01(
	ctx context.Context,
	domains []string,
	createRecord func(domain, fqdn, value string) error,
	deleteRecord func(domain, fqdn string),
) (*CertificateResult, error) {
	certKey, certKeyPEM, err := GenerateKey()
	if err != nil {
		return nil, err
	}

	order, err := c.acmeClient.AuthorizeOrder(ctx, acme.DomainIDs(domains...))
	if err != nil {
		return nil, fmt.Errorf("authorize order: %w", err)
	}

	for _, authURL := range order.AuthzURLs {
		auth, err := c.acmeClient.GetAuthorization(ctx, authURL)
		if err != nil {
			return nil, fmt.Errorf("get authorization: %w", err)
		}
		if auth.Status == acme.StatusValid {
			continue
		}

		var challenge *acme.Challenge
		for _, ch := range auth.Challenges {
			if ch.Type == "dns-01" {
				challenge = ch
				break
			}
		}
		if challenge == nil {
			return nil, fmt.Errorf("no DNS-01 challenge for %s", auth.Identifier.Value)
		}

		keyAuth, err := c.acmeClient.DNS01ChallengeRecord(challenge.Token)
		if err != nil {
			return nil, fmt.Errorf("DNS-01 record value: %w", err)
		}

		domain := auth.Identifier.Value
		fqdn := "_acme-challenge." + domain

		if err := createRecord(domain, fqdn, keyAuth); err != nil {
			return nil, fmt.Errorf("create DNS record for %s: %w", domain, err)
		}
		defer deleteRecord(domain, fqdn)

		// Wait for DNS propagation
		log.Printf("Waiting for DNS propagation of %s...", fqdn)
		time.Sleep(30 * time.Second)

		if _, err := c.acmeClient.Accept(ctx, challenge); err != nil {
			return nil, fmt.Errorf("accept challenge for %s: %w", domain, err)
		}
		if _, err := c.acmeClient.WaitAuthorization(ctx, authURL); err != nil {
			return nil, fmt.Errorf("wait authorization for %s: %w", domain, err)
		}
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		DNSNames: domains,
	}, certKey)
	if err != nil {
		return nil, fmt.Errorf("create CSR: %w", err)
	}

	derChain, certURL, err := c.acmeClient.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return nil, fmt.Errorf("finalize order: %w", err)
	}

	cert, err := x509.ParseCertificate(derChain[0])
	if err != nil {
		return nil, fmt.Errorf("parse cert: %w", err)
	}

	var certPEM, chainPEM string
	for i, der := range derChain {
		block := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		if i == 0 {
			certPEM = string(block)
		} else {
			chainPEM += string(block)
		}
	}

	return &CertificateResult{
		CertPEM:   certPEM + chainPEM,
		KeyPEM:    certKeyPEM,
		ChainPEM:  chainPEM,
		ExpiresAt: cert.NotAfter,
		IssuedAt:  cert.NotBefore,
		CertURL:   certURL,
	}, nil
}

// CloudflareProvider implements DNS record creation/deletion via Cloudflare API.
// Other providers follow the same pattern.
type CloudflareProvider struct {
	APIToken string
}

func NewCloudflareProvider(credentials map[string]string) *CloudflareProvider {
	return &CloudflareProvider{APIToken: credentials["api_token"]}
}

// Note: Real implementation would use Cloudflare API v4.
// For now this is a placeholder showing the interface pattern.
// In production, use a library like github.com/cloudflare/cloudflare-go
func (p *CloudflareProvider) CreateRecord(domain, fqdn, value string) error {
	log.Printf("Cloudflare: would create TXT record %s = %s for domain %s", fqdn, value, domain)
	// TODO: Implement Cloudflare API call
	// POST https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records
	return nil
}

func (p *CloudflareProvider) DeleteRecord(domain, fqdn string) {
	log.Printf("Cloudflare: would delete TXT record %s for domain %s", fqdn, domain)
	// TODO: Implement Cloudflare API call
}

// GetProvider returns create/delete functions for a DNS provider.
func GetProvider(config DNSProviderConfig) (
	createRecord func(domain, fqdn, value string) error,
	deleteRecord func(domain, fqdn string),
	err error,
) {
	switch config.Provider {
	case "cloudflare":
		p := NewCloudflareProvider(config.Credentials)
		return p.CreateRecord, p.DeleteRecord, nil
	// Add more providers here as needed
	default:
		return nil, nil, fmt.Errorf("unsupported DNS provider: %s", config.Provider)
	}
}

