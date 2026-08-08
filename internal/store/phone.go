package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) UpsertPhoneAssociation(ctx context.Context, value PhoneAssociation) error {
	value.ICCID = strings.TrimSpace(value.ICCID)
	value.DeviceID = strings.TrimSpace(value.DeviceID)
	value.Number = strings.TrimSpace(value.Number)
	value.Source = strings.TrimSpace(value.Source)
	if value.ICCID == "" || value.Number == "" || value.Source == "" {
		return errors.New("phone association ICCID, number, and source are required")
	}
	now := time.Now().UTC()
	createdAt := value.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := value.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO phone_associations (
			iccid, device_id, number, source, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(iccid) DO UPDATE SET
			device_id = excluded.device_id,
			number = excluded.number,
			source = excluded.source,
			updated_at = excluded.updated_at
	`,
		value.ICCID,
		value.DeviceID,
		value.Number,
		value.Source,
		createdAt.Unix(),
		updatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("upsert phone association for ICCID %q: %w", value.ICCID, err)
	}
	return nil
}

func (s *Store) PhoneAssociation(
	ctx context.Context,
	iccid string,
) (PhoneAssociation, error) {
	var value PhoneAssociation
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(ctx, `
		SELECT iccid, device_id, number, source, created_at, updated_at
		FROM phone_associations
		WHERE iccid = ?
	`, strings.TrimSpace(iccid)).Scan(
		&value.ICCID,
		&value.DeviceID,
		&value.Number,
		&value.Source,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PhoneAssociation{}, ErrNotFound
	}
	if err != nil {
		return PhoneAssociation{}, err
	}
	value.CreatedAt = time.Unix(createdAt, 0).UTC()
	value.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return value, nil
}
