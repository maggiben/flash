package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flash-reservation/backend/internal/clock"
	"github.com/flash-reservation/backend/internal/db"
	"github.com/flash-reservation/backend/internal/models"
	"github.com/flash-reservation/backend/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@postgres:5432/reservations?sslmode=disable"
	}
	return url
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("MIGRATIONS_DIR"); d != "" {
		return d
	}
	// go test sets cwd to the package directory; module root is two levels up.
	return "../../../migrations"
}

func resetDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		TRUNCATE idempotency_keys, reservations RESTART IDENTITY;
		DELETE FROM items`)
	return err
}

func setupService(t *testing.T) *service.ReservationService {
	t.Helper()
	ctx := context.Background()
	pool, err := db.NewPool(ctx, testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	require.NoError(t, db.RunMigrations(ctx, pool, migrationsDir(t)))
	require.NoError(t, resetDatabase(ctx, pool))

	svc := service.NewReservationService(pool, clock.System{}, 60)
	return svc
}

func TestConcurrentLastItem(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	itemID, err := svc.CreateTestItem(ctx, "last-unit-test", 1)
	require.NoError(t, err)

	const goroutines = 55
	var wg sync.WaitGroup
	var successCount atomic.Int32

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("last-item-key-%d", i)
			user := fmt.Sprintf("user-%d", i)
			_, status, err := svc.CreateReservation(ctx, user, key, models.CreateReservationRequest{
				ItemID: itemID, Quantity: 1,
			})
			if err == nil && (status == 201 || status == 200) {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int32(1), successCount.Load(), "exactly one reservation must succeed")

	total, reserved, err := svc.GetItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, 1, reserved)
	require.Equal(t, 0, total-reserved)
}

func TestConcurrentTenUnits(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	itemID, err := svc.CreateTestItem(ctx, "ten-units-test", 10)
	require.NoError(t, err)

	const goroutines = 100
	var wg sync.WaitGroup
	var successCount atomic.Int32

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("ten-units-key-%d", i)
			user := fmt.Sprintf("user-%d", i)
			_, status, err := svc.CreateReservation(ctx, user, key, models.CreateReservationRequest{
				ItemID: itemID, Quantity: 1,
			})
			if err == nil && (status == 201 || status == 200) {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int32(10), successCount.Load(), "exactly ten reservations must succeed")

	total, reserved, err := svc.GetItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 10, total)
	require.Equal(t, 10, reserved)
	require.GreaterOrEqual(t, total-reserved, 0)
}

func TestReserveIdempotencyParallel(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	itemID, err := svc.CreateTestItem(ctx, "idempotency-reserve", 5)
	require.NoError(t, err)

	const parallel = 2
	key := "same-idempotency-key"
	user := "idempotent-user"
	var wg sync.WaitGroup
	results := make([]models.Reservation, parallel)
	statuses := make([]int, parallel)
	errs := make([]error, parallel)

	wg.Add(parallel)
	for i := 0; i < parallel; i++ {
		i := i
		go func() {
			defer wg.Done()
			res, status, err := svc.CreateReservation(ctx, user, key, models.CreateReservationRequest{
				ItemID: itemID, Quantity: 1,
			})
			results[i] = res
			statuses[i] = status
			errs[i] = err
		}()
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, results[0].ID, results[1].ID, "same reservation id")

	activeCount, err := svc.CountActiveReservations(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 1, activeCount)

	_, reserved, err := svc.GetItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 1, reserved, "stock decremented exactly once")
}

func TestReleaseIdempotency(t *testing.T) {
	svc := setupService(t)
	ctx := context.Background()

	itemID, err := svc.CreateTestItem(ctx, "idempotency-release", 10)
	require.NoError(t, err)

	res, status, err := svc.CreateReservation(ctx, "release-user", "release-key-1", models.CreateReservationRequest{
		ItemID: itemID, Quantity: 3,
	})
	require.NoError(t, err)
	require.Equal(t, 201, status)

	_, reservedBefore, err := svc.GetItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 3, reservedBefore)

	r1, err := svc.ReleaseReservation(ctx, "release-user", res.ID)
	require.NoError(t, err)
	require.False(t, r1.Noop)

	r2, err := svc.ReleaseReservation(ctx, "release-user", res.ID)
	require.NoError(t, err)
	require.True(t, r2.Noop)

	_, reservedAfter, err := svc.GetItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 0, reservedAfter, "stock returned exactly once")
}

func TestExpirationReturnsStock(t *testing.T) {
	ctx := context.Background()
	pool, err := db.NewPool(ctx, testDatabaseURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })
	require.NoError(t, db.RunMigrations(ctx, pool, migrationsDir(t)))
	require.NoError(t, resetDatabase(ctx, pool))

	fixed := &clock.Fixed{T: time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)}
	svc := service.NewReservationService(pool, fixed, 60)
	expiration := service.NewExpirationService(pool, fixed)

	itemID, err := svc.CreateTestItem(ctx, "expire-test", 5)
	require.NoError(t, err)

	_, _, err = svc.CreateReservation(ctx, "expire-user", "expire-key", models.CreateReservationRequest{
		ItemID: itemID, Quantity: 2,
	})
	require.NoError(t, err)

	_, reserved, err := svc.GetItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 2, reserved)

	fixed.T = fixed.T.Add(61 * time.Second)
	require.NoError(t, expiration.ExpireOnce(ctx))

	_, reserved, err = svc.GetItem(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 0, reserved)

	active, err := svc.CountActiveReservations(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, 0, active)
}

// Ensure uuid import used in tests
var _ = uuid.Nil
