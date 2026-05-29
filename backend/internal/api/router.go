package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/flash-reservation/backend/internal/models"
	"github.com/flash-reservation/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type Server struct {
	reservations *service.ReservationService
	openAPIPath  string
}

func NewRouter(svc *service.ReservationService, openAPIPath string) http.Handler {
	s := &Server{reservations: svc, openAPIPath: openAPIPath}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/openapi.yaml", s.serveOpenAPI)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/inventory", s.listInventory)
		r.Route("/reservations", func(r chi.Router) {
			r.Get("/", s.listReservations)
			r.Post("/", s.createReservation)
			r.Delete("/{id}", s.releaseReservation)
		})
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Id, Idempotency-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(s.openAPIPath)
	if err != nil {
		writeError(w, http.StatusNotFound, models.ErrNotFound, "OpenAPI spec not found")
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) listInventory(w http.ResponseWriter, r *http.Request) {
	items, err := s.reservations.ListInventory(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) listReservations(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeError(w, http.StatusBadRequest, models.ErrMissingUserID, "X-User-Id header is required")
		return
	}
	list, err := s.reservations.ListActiveReservations(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	if list == nil {
		list = []models.Reservation{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"reservations": list})
}

func (s *Server) createReservation(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeError(w, http.StatusBadRequest, models.ErrMissingUserID, "X-User-Id header is required")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, models.ErrMissingIdempotencyKey, "Idempotency-Key header is required")
		return
	}

	var req models.CreateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrValidation, "invalid JSON body")
		return
	}

	reservation, status, err := s.reservations.CreateReservation(r.Context(), userID, idempotencyKey, req)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, status, reservation)
}

func (s *Server) releaseReservation(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeError(w, http.StatusBadRequest, models.ErrMissingUserID, "X-User-Id header is required")
		return
	}
	idStr := chi.URLParam(r, "id")
	resID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, models.ErrValidation, "invalid reservation id")
		return
	}

	result, err := s.reservations.ReleaseReservation(r.Context(), userID, resID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func mapServiceError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, models.ErrInsufficientStock):
		writeError(w, http.StatusConflict, models.ErrInsufficientStock, "Not enough stock available for this reservation")
	case strings.Contains(msg, models.ErrIdempotencyConflict):
		writeError(w, http.StatusConflict, models.ErrIdempotencyConflict, "Idempotency key was already used with a different request payload")
	case strings.Contains(msg, models.ErrNotFound):
		writeError(w, http.StatusNotFound, models.ErrNotFound, strings.TrimPrefix(msg, models.ErrNotFound+": "))
	case strings.Contains(msg, models.ErrForbidden):
		writeError(w, http.StatusForbidden, models.ErrForbidden, strings.TrimPrefix(msg, models.ErrForbidden+": "))
	case strings.Contains(msg, models.ErrValidation):
		writeError(w, http.StatusBadRequest, models.ErrValidation, strings.TrimPrefix(msg, models.ErrValidation+": "))
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL", msg)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorResponse{
		Error: models.APIError{Code: code, Message: message},
	})
}

// Ensure errors package used
var _ = errors.New
