package firewall

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("rule not found")

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Status() *StatusResponse {
	backend := "iptables"
	if _, err := exec.LookPath("nft"); err == nil {
		backend = "nftables"
	}

	enabled := false
	switch backend {
	case "iptables":
		out, err := exec.Command("iptables", "-L", "-n").CombinedOutput()
		if err == nil && len(out) > 0 {
			enabled = true
		}
	case "nftables":
		out, err := exec.Command("nft", "list", "ruleset").CombinedOutput()
		if err == nil && len(out) > 0 {
			enabled = true
		}
	}

	return &StatusResponse{Enabled: enabled, Backend: backend}
}

func (s *Service) ListRules(ctx context.Context) ([]Rule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, chain, protocol, src_ip, dst_port, action, description, position, created_at
		 FROM firewall_rules ORDER BY position, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.Chain, &r.Protocol, &r.SrcIP, &r.DstPort, &r.Action, &r.Description, &r.Position, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *Service) CreateRule(ctx context.Context, req CreateRuleRequest) (*Rule, error) {
	if req.Chain == "" {
		req.Chain = "INPUT"
	}
	if req.Action == "" {
		req.Action = "accept"
	}

	// Get next position
	var maxPos int
	s.pool.QueryRow(ctx, "SELECT COALESCE(MAX(position), 0) FROM firewall_rules").Scan(&maxPos)

	r := &Rule{
		Chain:       strings.ToUpper(req.Chain),
		Protocol:    strings.ToLower(req.Protocol),
		SrcIP:       req.SrcIP,
		DstPort:     req.DstPort,
		Action:      strings.ToLower(req.Action),
		Description: req.Description,
		Position:    maxPos + 1,
	}

	err := s.pool.QueryRow(ctx,
		`INSERT INTO firewall_rules (chain, protocol, src_ip, dst_port, action, description, position)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		r.Chain, r.Protocol, r.SrcIP, r.DstPort, r.Action, r.Description, r.Position,
	).Scan(&r.ID, &r.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Apply rule
	s.applyRule(r)

	return r, nil
}

func (s *Service) DeleteRule(ctx context.Context, id int64) error {
	var r Rule
	err := s.pool.QueryRow(ctx,
		`SELECT id, chain, protocol, src_ip, dst_port, action, description, position, created_at
		 FROM firewall_rules WHERE id = $1`, id,
	).Scan(&r.ID, &r.Chain, &r.Protocol, &r.SrcIP, &r.DstPort, &r.Action, &r.Description, &r.Position, &r.CreatedAt)
	if err == pgx.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, "DELETE FROM firewall_rules WHERE id = $1", id)
	if err != nil {
		return err
	}

	// Remove from iptables
	s.removeRule(&r)
	return nil
}

func (s *Service) applyRule(r *Rule) {
	action := strings.ToUpper(r.Action)
	if action == "ACCEPT" || action == "DROP" || action == "REJECT" {
		// OK
	} else {
		action = "ACCEPT"
	}

	args := []string{"-A", r.Chain, "-p", r.Protocol}
	if r.SrcIP != "" {
		args = append(args, "-s", r.SrcIP)
	}
	if r.DstPort != "" {
		args = append(args, "--dport", r.DstPort)
	}
	args = append(args, "-j", action)

	if r.Description != "" {
		args = append(args, "-m", "comment", "--comment", r.Description)
	}

	exec.Command("iptables", args...).Run()
}

func (s *Service) removeRule(r *Rule) {
	action := strings.ToUpper(r.Action)
	if action != "ACCEPT" && action != "DROP" && action != "REJECT" {
		action = "ACCEPT"
	}

	args := []string{"-D", r.Chain, "-p", r.Protocol}
	if r.SrcIP != "" {
		args = append(args, "-s", r.SrcIP)
	}
	if r.DstPort != "" {
		args = append(args, "--dport", r.DstPort)
	}
	args = append(args, "-j", action)

	exec.Command("iptables", args...).Run()
}

func (s *Service) ReapplyAll(ctx context.Context) error {
	rules, err := s.ListRules(ctx)
	if err != nil {
		return err
	}

	// Flush and reapply
	exec.Command("iptables", "-F").Run()

	// Allow loopback
	exec.Command("iptables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT").Run()
	// Allow established
	exec.Command("iptables", "-A", "INPUT", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT").Run()

	for i := range rules {
		s.applyRule(&rules[i])
	}

	fmt.Printf("Reapplied %d firewall rules\n", len(rules))
	return nil
}
