package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sette/guardian-lan/services/control-plane/internal/domain"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) UpsertDevice(ctx context.Context, device domain.Device) (domain.Device, bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM devices WHERE id = $1)", device.ID).Scan(&exists)
	if err != nil {
		return domain.Device{}, false, fmt.Errorf("check device existence: %w", err)
	}

	ipsJSON, err := json.Marshal(device.IPs)
	if err != nil {
		return domain.Device{}, false, fmt.Errorf("marshal ips: %w", err)
	}

	if exists {
		_, err = s.pool.Exec(ctx, `
			UPDATE devices
			SET mac = $2,
			    ips = $3::jsonb,
			    hostname = $4,
			    vendor = $5,
			    device_type = $6,
			    profile_id = $7,
			    managed = $8,
			    risk_score = $9,
			    last_seen_at = $10,
			    updated_at = NOW()
			WHERE id = $1
		`, device.ID, device.MAC, string(ipsJSON), device.Hostname, device.Vendor, device.DeviceType, device.ProfileID, device.Managed, device.RiskScore, device.LastSeen)
		if err != nil {
			return domain.Device{}, false, fmt.Errorf("update device: %w", err)
		}
	} else {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO devices (
				id, mac, ips, hostname, vendor, device_type, profile_id, managed, risk_score, first_seen_at, last_seen_at
			) VALUES (
				$1, $2, $3::jsonb, $4, $5, $6, $7, $8, $9, $10, $11
			)
		`, device.ID, device.MAC, string(ipsJSON), device.Hostname, device.Vendor, device.DeviceType, device.ProfileID, device.Managed, device.RiskScore, device.FirstSeen, device.LastSeen)
		if err != nil {
			return domain.Device{}, false, fmt.Errorf("insert device: %w", err)
		}
	}

	stored, err := s.GetDevice(ctx, device.ID)
	return stored, !exists, err
}

func (s *PostgresStore) GetDevice(ctx context.Context, id string) (domain.Device, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, mac, ips, hostname, vendor, device_type, profile_id, managed, risk_score, first_seen_at, last_seen_at
		FROM devices
		WHERE id = $1
	`, id)

	device, err := scanDevice(row)
	if err != nil {
		return domain.Device{}, err
	}

	return device, nil
}

func (s *PostgresStore) ListDevices(ctx context.Context) ([]domain.Device, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, mac, ips, hostname, vendor, device_type, profile_id, managed, risk_score, first_seen_at, last_seen_at
		FROM devices
		ORDER BY last_seen_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	return devices, rows.Err()
}

func (s *PostgresStore) UpdateDeviceProfile(ctx context.Context, id string, profileID string) (domain.Device, error) {
	commandTag, err := s.pool.Exec(ctx, "UPDATE devices SET profile_id = $2, updated_at = NOW() WHERE id = $1", id, profileID)
	if err != nil {
		return domain.Device{}, fmt.Errorf("update profile: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.Device{}, pgx.ErrNoRows
	}

	return s.GetDevice(ctx, id)
}

func (s *PostgresStore) StoreDNSEvent(ctx context.Context, event domain.DNSEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO dns_events (id, device_id, query, domain, category, resolver, blocked, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.DeviceID, event.Query, event.Domain, event.Category, event.Resolver, event.Blocked, event.ObservedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("store dns event: %w", err)
	}

	return nil
}

func (s *PostgresStore) ListDNSEvents(ctx context.Context, limit int) ([]domain.DNSEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, device_id, query, domain, category, resolver, blocked, observed_at
		FROM dns_events
		ORDER BY observed_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list dns events: %w", err)
	}
	defer rows.Close()

	var events []domain.DNSEvent
	for rows.Next() {
		var event domain.DNSEvent
		if err := rows.Scan(&event.ID, &event.DeviceID, &event.Query, &event.Domain, &event.Category, &event.Resolver, &event.Blocked, &event.ObservedAt); err != nil {
			return nil, fmt.Errorf("scan dns event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (s *PostgresStore) StoreFlowEvent(ctx context.Context, event domain.FlowEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO flow_events (id, device_id, src_ip, dst_ip, dst_port, protocol, bytes_in, bytes_out, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.ID, event.DeviceID, event.SrcIP, event.DstIP, event.DstPort, event.Protocol, event.BytesIn, event.BytesOut, event.ObservedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("store flow event: %w", err)
	}

	return nil
}

func (s *PostgresStore) ListFlowEvents(ctx context.Context, limit int) ([]domain.FlowEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, device_id, src_ip, dst_ip, dst_port, protocol, bytes_in, bytes_out, observed_at
		FROM flow_events
		ORDER BY observed_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list flow events: %w", err)
	}
	defer rows.Close()

	var events []domain.FlowEvent
	for rows.Next() {
		var event domain.FlowEvent
		if err := rows.Scan(&event.ID, &event.DeviceID, &event.SrcIP, &event.DstIP, &event.DstPort, &event.Protocol, &event.BytesIn, &event.BytesOut, &event.ObservedAt); err != nil {
			return nil, fmt.Errorf("scan flow event: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (s *PostgresStore) StoreObservation(ctx context.Context, observation domain.Observation) error {
	evidenceJSON, err := json.Marshal(observation.EvidenceRef)
	if err != nil {
		return fmt.Errorf("marshal observation evidence: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO observations (id, device_id, source, kind, severity, summary, evidence_ref, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
	`, observation.ID, observation.DeviceID, observation.Source, observation.Kind, observation.Severity, observation.Summary, string(evidenceJSON), observation.ObservedAt)
	if err != nil {
		return fmt.Errorf("store observation: %w", err)
	}

	return nil
}

func (s *PostgresStore) CreateAlert(ctx context.Context, alert domain.Alert) error {
	evidenceJSON, err := json.Marshal(alert.Evidence)
	if err != nil {
		return fmt.Errorf("marshal alert evidence: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO alerts (id, device_id, type, severity, title, status, evidence, created_at, acknowledged_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)
	`, alert.ID, alert.DeviceID, alert.Type, alert.Severity, alert.Title, alert.Status, string(evidenceJSON), alert.CreatedAt, alert.Acknowledged)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("create alert: %w", err)
	}

	return nil
}

func (s *PostgresStore) ListAlerts(ctx context.Context, limit int, status string) ([]domain.Alert, error) {
	query := `
		SELECT id, device_id, type, severity, title, status, evidence, created_at, acknowledged_at
		FROM alerts
	`
	args := []any{}
	if status != "" {
		query += " WHERE status = $1"
		args = append(args, status)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []domain.Alert
	for rows.Next() {
		alert, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

func (s *PostgresStore) AckAlert(ctx context.Context, id string) (domain.Alert, error) {
	ackTime := time.Now().UTC()
	commandTag, err := s.pool.Exec(ctx, `
		UPDATE alerts
		SET status = 'acknowledged',
		    acknowledged_at = $2
		WHERE id = $1
	`, id, ackTime)
	if err != nil {
		return domain.Alert{}, fmt.Errorf("ack alert: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.Alert{}, pgx.ErrNoRows
	}

	row := s.pool.QueryRow(ctx, `
		SELECT id, device_id, type, severity, title, status, evidence, created_at, acknowledged_at
		FROM alerts
		WHERE id = $1
	`, id)

	return scanAlert(row)
}

func scanDevice(scanner interface {
	Scan(...any) error
}) (domain.Device, error) {
	var device domain.Device
	var ipsJSON []byte
	err := scanner.Scan(&device.ID, &device.MAC, &ipsJSON, &device.Hostname, &device.Vendor, &device.DeviceType, &device.ProfileID, &device.Managed, &device.RiskScore, &device.FirstSeen, &device.LastSeen)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Device{}, err
		}
		return domain.Device{}, fmt.Errorf("scan device: %w", err)
	}

	if err := json.Unmarshal(ipsJSON, &device.IPs); err != nil {
		return domain.Device{}, fmt.Errorf("unmarshal device ips: %w", err)
	}

	return device, nil
}

func scanAlert(scanner interface {
	Scan(...any) error
}) (domain.Alert, error) {
	var alert domain.Alert
	var evidenceJSON []byte
	err := scanner.Scan(&alert.ID, &alert.DeviceID, &alert.Type, &alert.Severity, &alert.Title, &alert.Status, &evidenceJSON, &alert.CreatedAt, &alert.Acknowledged)
	if err != nil {
		return domain.Alert{}, fmt.Errorf("scan alert: %w", err)
	}

	if len(evidenceJSON) == 0 {
		alert.Evidence = map[string]any{}
		return alert, nil
	}

	if err := json.Unmarshal(evidenceJSON, &alert.Evidence); err != nil {
		return domain.Alert{}, fmt.Errorf("unmarshal alert evidence: %w", err)
	}

	return alert, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
