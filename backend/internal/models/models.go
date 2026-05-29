package models

import (
	"time"

	"github.com/google/uuid"
)

type InventoryItem struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	TotalQuantity    int       `json:"total_quantity"`
	ReservedQuantity int       `json:"reserved_quantity"`
	AvailableQuantity int      `json:"available_quantity"`
}

type Reservation struct {
	ID        uuid.UUID `json:"id"`
	ItemID    uuid.UUID `json:"item_id"`
	ItemName  string    `json:"item_name"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type ReleaseResult struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
	Noop   bool      `json:"noop"`
}

type CreateReservationRequest struct {
	ItemID   uuid.UUID `json:"item_id"`
	Quantity int       `json:"quantity"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

const (
	StatusActive    = "active"
	StatusConfirmed = "confirmed"
	StatusReleased  = "released"
	StatusExpired   = "expired"
)

const (
	ErrInsufficientStock    = "INSUFFICIENT_STOCK"
	ErrIdempotencyConflict  = "IDEMPOTENCY_CONFLICT"
	ErrNotFound             = "NOT_FOUND"
	ErrForbidden            = "FORBIDDEN"
	ErrValidation           = "VALIDATION_ERROR"
	ErrMissingUserID        = "MISSING_USER_ID"
	ErrMissingIdempotencyKey = "MISSING_IDEMPOTENCY_KEY"
	ErrInvalidState          = "INVALID_STATE"
)
