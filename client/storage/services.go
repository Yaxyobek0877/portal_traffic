package storage

import (
	"fmt"
	"time"
)

// ExposedService is a single saved service-expose preference. Persisted
// across launches so the user re-enters a portal with their cameras /
// game servers / dev URLs already shared, without redoing the form.
type ExposedService struct {
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"` // "tcp" | "udp"
	Name      string `json:"name"`
	Target    string `json:"target"`  // "" → 127.0.0.1:<port>
	Enabled   bool   `json:"enabled"` // false = remembered but skipped on auto-restore
	// RequireApproval gates every peer dial attempt: when true, the
	// host's UI is asked to allow/deny each new (peer, port) combo
	// before the proxy connects to the target. Once approved for a
	// given peer, the decision is cached for the session so the user
	// isn't re-prompted on every reconnect. False = auto-allow (the
	// behaviour from before this flag).
	RequireApproval bool  `json:"requireApproval"`
	UpdatedAt       int64 `json:"updatedAt"`
}

// SaveExposedService upserts the row keyed on (port, protocol). Setting
// Enabled=false keeps the row but won't be auto-applied — handy when
// the user wants to temporarily stop sharing a camera without losing
// the LAN-target IP they typed.
func (s *Store) SaveExposedService(svc ExposedService) error {
	if svc.UpdatedAt == 0 {
		svc.UpdatedAt = time.Now().Unix()
	}
	enabled := 0
	if svc.Enabled {
		enabled = 1
	}
	approval := 0
	if svc.RequireApproval {
		approval = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO exposed_services (port, protocol, name, target, enabled, require_approval, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(port, protocol) DO UPDATE SET
			name=excluded.name,
			target=excluded.target,
			enabled=excluded.enabled,
			require_approval=excluded.require_approval,
			updated_at=excluded.updated_at
	`, svc.Port, svc.Protocol, svc.Name, svc.Target, enabled, approval, svc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("storage: save exposed service: %w", err)
	}
	return nil
}

// DeleteExposedService removes the saved entry entirely. Use this when
// the user clicks the trash icon — they've decided they don't want this
// service shared next session either.
func (s *Store) DeleteExposedService(port int, protocol string) error {
	_, err := s.db.Exec(
		`DELETE FROM exposed_services WHERE port=? AND protocol=?`,
		port, protocol,
	)
	return err
}

// ListExposedServices returns every saved row. Caller filters by
// Enabled themselves.
func (s *Store) ListExposedServices() ([]ExposedService, error) {
	rows, err := s.db.Query(`
		SELECT port, protocol, name, target, enabled, require_approval, updated_at
		FROM exposed_services
		ORDER BY port, protocol
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExposedService
	for rows.Next() {
		var svc ExposedService
		var enabled, approval int
		if err := rows.Scan(
			&svc.Port, &svc.Protocol, &svc.Name, &svc.Target,
			&enabled, &approval, &svc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		svc.Enabled = enabled != 0
		svc.RequireApproval = approval != 0
		out = append(out, svc)
	}
	return out, rows.Err()
}
