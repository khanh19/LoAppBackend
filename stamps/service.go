package stamps

import (
	"context"

	"encore.app/pkg/externaldb"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//encore:service
type Service struct {
	db *pgxpool.Pool
}

func initService() (*Service, error) {
	db, err := externaldb.Get(context.Background())
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to initialize stamps database")
	}
	return &Service{db: db}, nil
}

var secrets struct {
	// SupabaseStorageServiceKey is the service-role key for signed upload URLs.
	SupabaseStorageServiceKey string
	// SupabaseURL is the project URL, e.g. https://xofbxjbrzpsocnjdsmmk.supabase.co
	SupabaseURL string
}
