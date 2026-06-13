package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func pgUUID(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }

func sameUUID(p pgtype.UUID, id uuid.UUID) bool { return p.Valid && p.Bytes == id }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// randomToken returns a URL-safe random secret of n bytes.
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// deviceView is the device shape returned to clients (no secrets).
type deviceView struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	SerialNumber    *string    `json:"serial_number"`
	FirmwareVersion string     `json:"firmware_version"`
	LastSeenAt      *time.Time `json:"last_seen_at"`
	ClaimedAt       *time.Time `json:"claimed_at"`
}

func toDeviceView(d store.Device) deviceView {
	v := deviceView{
		ID:              d.ID,
		Name:            d.Name,
		SerialNumber:    d.SerialNumber,
		FirmwareVersion: d.FirmwareVersion,
	}
	if d.LastSeenAt.Valid {
		t := d.LastSeenAt.Time
		v.LastSeenAt = &t
	}
	if d.ClaimedAt.Valid {
		t := d.ClaimedAt.Time
		v.ClaimedAt = &t
	}
	return v
}
