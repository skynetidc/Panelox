package dns

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrZoneExists    = errors.New("zone already exists")
	ErrInvalidDomain = errors.New("invalid domain name")
	ErrInvalidType   = errors.New("invalid record type")

	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Zones

func (s *Service) CreateZone(ctx context.Context, userID int64, req CreateZoneRequest) (*Zone, error) {
	name := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(req.Name)), ".")
	if !domainRegex.MatchString(name) {
		return nil, ErrInvalidDomain
	}

	existing, err := s.repo.GetZoneByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrZoneExists
	}

	z := &Zone{
		UserID: userID,
		Name:   name,
		Status: "active",
		Serial: 1,
	}
	if err := s.repo.CreateZone(ctx, z); err != nil {
		return nil, err
	}

	// Create default SOA-related records
	defaults := []Record{
		{ZoneID: z.ID, Name: "@", Type: "NS", Content: "ns1." + name + ".", TTL: 86400},
		{ZoneID: z.ID, Name: "@", Type: "NS", Content: "ns2." + name + ".", TTL: 86400},
		{ZoneID: z.ID, Name: "@", Type: "A", Content: "0.0.0.0", TTL: 3600},
	}
	for i := range defaults {
		s.repo.CreateRecord(ctx, &defaults[i])
	}

	z.Records, _ = s.repo.ListRecords(ctx, z.ID)
	return z, nil
}

func (s *Service) GetZone(ctx context.Context, id int64) (*Zone, error) {
	z, err := s.repo.GetZone(ctx, id)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrNotFound
	}
	z.Records, _ = s.repo.ListRecords(ctx, z.ID)
	return z, nil
}

func (s *Service) ListZones(ctx context.Context, page, size int) ([]Zone, int64, error) {
	zones, total, err := s.repo.ListZones(ctx, page, size)
	if err != nil {
		return nil, 0, err
	}
	// Attach record counts
	for i := range zones {
		count, _ := s.repo.RecordCount(ctx, zones[i].ID)
		zones[i].Records = make([]Record, count) // just for count in JSON
	}
	return zones, total, err
}

func (s *Service) DeleteZone(ctx context.Context, id int64) error {
	z, err := s.repo.GetZone(ctx, id)
	if err != nil {
		return err
	}
	if z == nil {
		return ErrNotFound
	}
	return s.repo.DeleteZone(ctx, id) // CASCADE deletes records
}

// Records

func (s *Service) CreateRecord(ctx context.Context, zoneID int64, req CreateRecordRequest) (*Record, error) {
	z, err := s.repo.GetZone(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrNotFound
	}

	if !isValidRecordType(req.Type) {
		return nil, ErrInvalidType
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 3600
	}

	rec := &Record{
		ZoneID:   zoneID,
		Name:     req.Name,
		Type:     strings.ToUpper(req.Type),
		Content:  req.Content,
		TTL:      ttl,
		Priority: req.Priority,
	}

	if err := s.repo.CreateRecord(ctx, rec); err != nil {
		return nil, err
	}

	s.repo.IncrementSerial(ctx, zoneID)
	return rec, nil
}

func (s *Service) UpdateRecord(ctx context.Context, id int64, req UpdateRecordRequest) error {
	rec, err := s.repo.GetRecord(ctx, id)
	if err != nil {
		return err
	}
	if rec == nil {
		return ErrNotFound
	}

	if req.Type != nil && !isValidRecordType(*req.Type) {
		return ErrInvalidType
	}
	if req.Type != nil {
		upper := strings.ToUpper(*req.Type)
		req.Type = &upper
	}

	if err := s.repo.UpdateRecord(ctx, id, req); err != nil {
		return err
	}

	s.repo.IncrementSerial(ctx, rec.ZoneID)
	return nil
}

func (s *Service) DeleteRecord(ctx context.Context, id int64) error {
	rec, err := s.repo.GetRecord(ctx, id)
	if err != nil {
		return err
	}
	if rec == nil {
		return ErrNotFound
	}

	if err := s.repo.DeleteRecord(ctx, id); err != nil {
		return err
	}

	s.repo.IncrementSerial(ctx, rec.ZoneID)
	return nil
}

func (s *Service) ListRecords(ctx context.Context, zoneID int64) ([]Record, error) {
	return s.repo.ListRecords(ctx, zoneID)
}

// GenerateZoneFile generates a BIND-style zone file for export.
func (s *Service) GenerateZoneFile(ctx context.Context, zoneID int64) (string, error) {
	z, err := s.repo.GetZone(ctx, zoneID)
	if err != nil {
		return "", err
	}
	if z == nil {
		return "", ErrNotFound
	}

	records, err := s.repo.ListRecords(ctx, zoneID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("; Zone file for %s\n", z.Name))
	b.WriteString(fmt.Sprintf("; Serial: %d\n", z.Serial))
	b.WriteString(fmt.Sprintf("$ORIGIN %s.\n", z.Name))
	b.WriteString("$TTL 3600\n\n")

	// SOA record
	b.WriteString(fmt.Sprintf("@ IN SOA ns1.%s. admin.%s. (\n", z.Name, z.Name))
	b.WriteString(fmt.Sprintf("    %d  ; serial\n", z.Serial))
	b.WriteString("    3600    ; refresh\n")
	b.WriteString("    900     ; retry\n")
	b.WriteString("    1209600 ; expire\n")
	b.WriteString("    86400   ; minimum TTL\n")
	b.WriteString(")\n\n")

	for _, rec := range records {
		name := rec.Name
		if name == "" {
			name = "@"
		}
		if rec.Priority != nil && (rec.Type == "MX" || rec.Type == "SRV") {
			b.WriteString(fmt.Sprintf("%-24s %d IN %-6s %d %s\n", name, rec.TTL, rec.Type, *rec.Priority, rec.Content))
		} else {
			b.WriteString(fmt.Sprintf("%-24s %d IN %-6s %s\n", name, rec.TTL, rec.Type, rec.Content))
		}
	}

	return b.String(), nil
}

func isValidRecordType(t string) bool {
	t = strings.ToUpper(t)
	for _, valid := range ValidRecordTypes {
		if t == valid {
			return true
		}
	}
	return false
}
