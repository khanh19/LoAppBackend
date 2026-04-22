package externaldb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go4.org/syncutil"
)

// Get returns a database connection pool to the external database.
// It is lazily created on first use.
func Get(ctx context.Context) (*pgxpool.Pool, error) {
	// Attempt to setup the database connection pool if it hasn't
	// already been successfully setup.
	err := once.Do(func() error {
		var err error
		pool, err = setup(ctx)
		return err
	})
	return pool, err
}

var (
	// once is like sync.Once except it re-arms itself on failure
	once syncutil.Once
	// pool is the successfully created database connection pool,
	// or nil when no such pool has been setup yet.
	pool *pgxpool.Pool
)

var secrets struct {
	// ExternalDBPassword: encore secret or .secrets.local.cue (app root).
	ExternalDBPassword string
}

// setup attempts to set up a database connection pool.
func setup(ctx context.Context) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf("postgresql://postgres:%s@db.xofbxjbrzpsocnjdsmmk.supabase.co:5432/postgres", secrets.ExternalDBPassword)
	return pgxpool.New(ctx, connString)
}
