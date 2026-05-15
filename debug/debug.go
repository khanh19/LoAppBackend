package debug

import (
	"context"

	"encore.app/pkg/externaldb"
)

// DBPing checks that the external database pool is reachable.
//
//encore:api public method=GET path=/debug/db-ping
func DBPing(ctx context.Context) (*DBPingResponse, error) {
	pool, err := externaldb.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &DBPingResponse{OK: true, Message: "external database reached"}, nil
}
