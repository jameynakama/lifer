package api

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jameynakama/flockdeck/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func TestInTx_RollsBackOnError(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code = '_txtest'`)
	})

	err := h.inTx(ctx, func(q store.Querier) error {
		if _, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
			EbirdCode: "_txtest", CommonName: "Tx Test", ScientificName: "Txus testus",
		}); err != nil {
			return err
		}
		return errors.New("boom")
	})

	require.Error(t, err)
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM species WHERE ebird_code = '_txtest'`).Scan(&n))
	assert.Equal(t, 0, n, "write inside a failed tx must roll back")
}

func TestInTx_CommitsOnSuccess(t *testing.T) {
	pool := connectTestPool(t)
	h := &Handler{queries: store.New(pool), db: pool}
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM species WHERE ebird_code = '_txtest2'`)
	})

	err := h.inTx(ctx, func(q store.Querier) error {
		_, err := q.UpsertSpecies(ctx, store.UpsertSpeciesParams{
			EbirdCode: "_txtest2", CommonName: "Tx Test", ScientificName: "Txus testus",
		})
		return err
	})

	require.NoError(t, err)
	var n int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM species WHERE ebird_code = '_txtest2'`).Scan(&n))
	assert.Equal(t, 1, n)
}

func TestInTx_NoPool_FallsBackToPlainQuerier(t *testing.T) {
	// Unit-test configuration: no pool, stub querier -- fn runs non-atomically.
	called := false
	h := makeHandler(&stubQuerier{})

	err := h.inTx(context.Background(), func(q store.Querier) error {
		called = true
		assert.NotNil(t, q)
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
}
