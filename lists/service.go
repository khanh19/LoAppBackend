package lists

import (
	"context"

	"encore.app/pkg/externaldb"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//encore:service
type Service struct {
	db     *pgxpool.Pool
	google *googlePlacesClient
}

func initService() (*Service, error) {
	db, err := externaldb.Get(context.Background())
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to initialize lists database")
	}
	return &Service{
		db:     db,
		google: newGooglePlacesClient(),
	}, nil
}
