package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flash-reservation/backend/internal/api"
	"github.com/flash-reservation/backend/internal/clock"
	"github.com/flash-reservation/backend/internal/db"
	"github.com/flash-reservation/backend/internal/service"
)

func main() {
	ctx := context.Background()

	databaseURL := envOr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/reservations?sslmode=disable")
	port := envOr("PORT", "8080")
	ttlSeconds := 60
	expireInterval := 5 * time.Second

	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("database pool: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	if err := db.RunSeed(ctx, pool, "seeds/seed.sql"); err != nil {
		log.Fatalf("seed: %v", err)
	}

	sysClock := clock.System{}
	reservationSvc := service.NewReservationService(pool, sysClock, ttlSeconds)
	expirationSvc := service.NewExpirationService(pool, sysClock)

	expireCtx, cancelExpire := context.WithCancel(ctx)
	defer cancelExpire()
	go expirationSvc.Run(expireCtx, expireInterval)

	router := api.NewRouter(reservationSvc, "specs/001-inventory-reservation/contracts/openapi.yaml")

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("API listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cancelExpire()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
