package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flash-reservation/backend/internal/clock"
	"github.com/flash-reservation/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReservationService struct {
	pool       *pgxpool.Pool
	clock      clock.Clock
	ttlSeconds int
}

func NewReservationService(pool *pgxpool.Pool, c clock.Clock, ttlSeconds int) *ReservationService {
	return &ReservationService{pool: pool, clock: c, ttlSeconds: ttlSeconds}
}

func (s *ReservationService) ListInventory(ctx context.Context) ([]models.InventoryItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, total_quantity, reserved_quantity,
		       total_quantity - reserved_quantity AS available_quantity
		FROM items
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.InventoryItem
	for rows.Next() {
		var item models.InventoryItem
		if err := rows.Scan(&item.ID, &item.Name, &item.TotalQuantity, &item.ReservedQuantity, &item.AvailableQuantity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *ReservationService) ListActiveReservations(ctx context.Context, userID string) ([]models.Reservation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.item_id, i.name, r.quantity, r.status, r.expires_at, r.created_at
		FROM reservations r
		JOIN items i ON i.id = r.item_id
		WHERE r.user_id = $1 AND r.status IN ('active', 'confirmed')
		ORDER BY r.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Reservation
	for rows.Next() {
		var r models.Reservation
		if err := rows.Scan(&r.ID, &r.ItemID, &r.ItemName, &r.Quantity, &r.Status, &r.ExpiresAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func hashRequest(itemID uuid.UUID, quantity int) string {
	payload := struct {
		ItemID   string `json:"item_id"`
		Quantity int    `json:"quantity"`
	}{ItemID: itemID.String(), Quantity: quantity}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *ReservationService) CreateReservation(ctx context.Context, userID, idempotencyKey string, req models.CreateReservationRequest) (models.Reservation, int, error) {
	if req.Quantity < 1 {
		return models.Reservation{}, 0, fmt.Errorf("%s: quantity must be at least 1", models.ErrValidation)
	}

	reqHash := hashRequest(req.ItemID, req.Quantity)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Reservation{}, 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, idempotencyKey); err != nil {
		return models.Reservation{}, 0, err
	}

	var existingStatus int
	var existingBody []byte
	var storedHash string
	err = tx.QueryRow(ctx, `
		SELECT request_hash, response_status, response_body
		FROM idempotency_keys
		WHERE key = $1`, idempotencyKey).Scan(&storedHash, &existingStatus, &existingBody)
	if err == nil {
		if storedHash != reqHash {
			return models.Reservation{}, 0, fmt.Errorf("%s: idempotency key reused with different payload", models.ErrIdempotencyConflict)
		}
		var reservation models.Reservation
		if err := json.Unmarshal(existingBody, &reservation); err != nil {
			return models.Reservation{}, 0, err
		}
		if err := tx.Commit(ctx); err != nil {
			return models.Reservation{}, 0, err
		}
		if existingStatus == 201 {
			return reservation, 200, nil
		}
		return reservation, existingStatus, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.Reservation{}, 0, err
	}

	var itemName string
	var totalQty, reservedQty int
	err = tx.QueryRow(ctx, `
		SELECT name, total_quantity, reserved_quantity
		FROM items WHERE id = $1 FOR UPDATE`, req.ItemID).Scan(&itemName, &totalQty, &reservedQty)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Reservation{}, 0, fmt.Errorf("%s: item not found", models.ErrNotFound)
	}
	if err != nil {
		return models.Reservation{}, 0, err
	}

	tag, err := tx.Exec(ctx, `
		UPDATE items SET reserved_quantity = reserved_quantity + $1
		WHERE id = $2 AND total_quantity - reserved_quantity >= $1`, req.Quantity, req.ItemID)
	if err != nil {
		return models.Reservation{}, 0, err
	}
	if tag.RowsAffected() == 0 {
		return models.Reservation{}, 0, fmt.Errorf("%s: not enough stock available", models.ErrInsufficientStock)
	}

	now := s.clock.Now()
	expiresAt := now.Add(time.Duration(s.ttlSeconds) * time.Second)
	resID := uuid.New()

	var reservation models.Reservation
	err = tx.QueryRow(ctx, `
		INSERT INTO reservations (id, item_id, user_id, quantity, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'active', $5, $6, $6)
		RETURNING id, item_id, quantity, status, expires_at, created_at`,
		resID, req.ItemID, userID, req.Quantity, expiresAt, now,
	).Scan(&reservation.ID, &reservation.ItemID, &reservation.Quantity, &reservation.Status, &reservation.ExpiresAt, &reservation.CreatedAt)
	if err != nil {
		return models.Reservation{}, 0, err
	}
	reservation.ItemName = itemName

	body, err := json.Marshal(reservation)
	if err != nil {
		return models.Reservation{}, 0, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_keys (key, request_hash, response_status, response_body, reservation_id)
		VALUES ($1, $2, 201, $3, $4)`,
		idempotencyKey, reqHash, body, reservation.ID)
	if err != nil {
		return models.Reservation{}, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Reservation{}, 0, err
	}
	return reservation, 201, nil
}

func (s *ReservationService) ReleaseReservation(ctx context.Context, userID string, reservationID uuid.UUID) (models.ReleaseResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.ReleaseResult{}, err
	}
	defer tx.Rollback(ctx)

	var ownerID, status string
	var itemID uuid.UUID
	var quantity int
	err = tx.QueryRow(ctx, `
		SELECT user_id, status, item_id, quantity
		FROM reservations WHERE id = $1 FOR UPDATE`, reservationID).Scan(&ownerID, &status, &itemID, &quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ReleaseResult{}, fmt.Errorf("%s: reservation not found", models.ErrNotFound)
	}
	if err != nil {
		return models.ReleaseResult{}, err
	}
	if ownerID != userID {
		return models.ReleaseResult{}, fmt.Errorf("%s: not your reservation", models.ErrForbidden)
	}

	if status != models.StatusActive && status != models.StatusConfirmed {
		if err := tx.Commit(ctx); err != nil {
			return models.ReleaseResult{}, err
		}
		return models.ReleaseResult{ID: reservationID, Status: status, Noop: true}, nil
	}

	tag, err := tx.Exec(ctx, `
		UPDATE reservations SET status = 'released', updated_at = $2
		WHERE id = $1 AND status IN ('active', 'confirmed')`, reservationID, s.clock.Now())
	if err != nil {
		return models.ReleaseResult{}, err
	}
	if tag.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return models.ReleaseResult{}, err
		}
		return models.ReleaseResult{ID: reservationID, Status: models.StatusReleased, Noop: true}, nil
	}

	_, err = tx.Exec(ctx, `
		UPDATE items SET reserved_quantity = reserved_quantity - $1
		WHERE id = $2`, quantity, itemID)
	if err != nil {
		return models.ReleaseResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.ReleaseResult{}, err
	}
	return models.ReleaseResult{ID: reservationID, Status: models.StatusReleased, Noop: false}, nil
}

func (s *ReservationService) ConfirmReservation(ctx context.Context, userID string, reservationID uuid.UUID) (models.Reservation, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Reservation{}, false, err
	}
	defer tx.Rollback(ctx)

	var ownerID, status, itemName string
	var itemID uuid.UUID
	var quantity int
	var expiresAt, createdAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT r.user_id, r.status, r.item_id, i.name, r.quantity, r.expires_at, r.created_at
		FROM reservations r
		JOIN items i ON i.id = r.item_id
		WHERE r.id = $1 FOR UPDATE`, reservationID).Scan(
		&ownerID, &status, &itemID, &itemName, &quantity, &expiresAt, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Reservation{}, false, fmt.Errorf("%s: reservation not found", models.ErrNotFound)
	}
	if err != nil {
		return models.Reservation{}, false, err
	}
	if ownerID != userID {
		return models.Reservation{}, false, fmt.Errorf("%s: not your reservation", models.ErrForbidden)
	}

	if status == models.StatusConfirmed {
		if err := tx.Commit(ctx); err != nil {
			return models.Reservation{}, false, err
		}
		return models.Reservation{
			ID: reservationID, ItemID: itemID, ItemName: itemName, Quantity: quantity,
			Status: models.StatusConfirmed, ExpiresAt: expiresAt, CreatedAt: createdAt,
		}, true, nil
	}

	if status != models.StatusActive {
		return models.Reservation{}, false, fmt.Errorf("%s: only active reservations can be confirmed (current: %s)", models.ErrInvalidState, status)
	}

	now := s.clock.Now()
	err = tx.QueryRow(ctx, `
		UPDATE reservations SET status = 'confirmed', updated_at = $2
		WHERE id = $1 AND status = 'active'
		RETURNING expires_at`,
		reservationID, now).Scan(&expiresAt)
	if err != nil {
		return models.Reservation{}, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Reservation{}, false, err
	}

	return models.Reservation{
		ID: reservationID, ItemID: itemID, ItemName: itemName, Quantity: quantity,
		Status: models.StatusConfirmed, ExpiresAt: expiresAt, CreatedAt: createdAt,
	}, false, nil
}

// ExpirationService expires unconfirmed (active) reservations past TTL.
type ExpirationService struct {
	pool  *pgxpool.Pool
	clock clock.Clock
}

func NewExpirationService(pool *pgxpool.Pool, c clock.Clock) *ExpirationService {
	return &ExpirationService{pool: pool, clock: c}
}

func (s *ExpirationService) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.ExpireOnce(ctx)
		}
	}
}

func (s *ExpirationService) ExpireOnce(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		UPDATE reservations SET status = 'expired', updated_at = $1
		WHERE status = 'active' AND expires_at <= $1
		RETURNING id, item_id, quantity`, s.clock.Now())
	if err != nil {
		return err
	}

	type expiredRow struct {
		id     uuid.UUID
		itemID uuid.UUID
		qty    int
	}
	var expired []expiredRow
	for rows.Next() {
		var row expiredRow
		if err := rows.Scan(&row.id, &row.itemID, &row.qty); err != nil {
			rows.Close()
			return err
		}
		expired = append(expired, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, row := range expired {
		if _, err := tx.Exec(ctx, `
			UPDATE items SET reserved_quantity = reserved_quantity - $1
			WHERE id = $2`, row.qty, row.itemID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Test helpers used by integration tests
func (s *ReservationService) Pool() *pgxpool.Pool { return s.pool }

func (s *ReservationService) ResetItemStock(ctx context.Context, itemID uuid.UUID, total int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE items SET total_quantity = $2, reserved_quantity = 0 WHERE id = $1`, itemID, total)
	return err
}

func (s *ReservationService) GetItem(ctx context.Context, itemID uuid.UUID) (total, reserved int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT total_quantity, reserved_quantity FROM items WHERE id = $1`, itemID).Scan(&total, &reserved)
	return
}

func (s *ReservationService) CreateTestItem(ctx context.Context, name string, total int) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO items (id, name, total_quantity, reserved_quantity) VALUES ($1, $2, $3, 0)`, id, name, total)
	return id, err
}

func (s *ReservationService) CountActiveReservations(ctx context.Context, itemID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM reservations WHERE item_id = $1 AND status = 'active'`, itemID).Scan(&n)
	return n, err
}

func (s *ReservationService) GetReservationStatus(ctx context.Context, id uuid.UUID) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `SELECT status FROM reservations WHERE id = $1`, id).Scan(&status)
	return status, err
}
