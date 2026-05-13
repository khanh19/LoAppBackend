package users

import (
	"context"
	"encoding/json"

	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
)

func (s *Service) upsertFromAuth(ctx context.Context, req *UpsertFromAuthParams, rawProfile json.RawMessage) (*User, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	user, err := findByIdentity(ctx, tx, req.Provider, req.ProviderUserID)
	if err != nil {
		return nil, err
	}
	isNewIdentity := user == nil

	if user == nil && req.Email != nil {
		user, err = findByEmail(ctx, tx, *req.Email)
		if err != nil {
			return nil, err
		}
	}
	if user == nil {
		user, err = createUser(ctx, tx, req.Email, req.EmailVerified)
		if err != nil {
			return nil, err
		}
	}

	if err := upsertIdentity(ctx, tx, user.ID, req, rawProfile); err != nil {
		return nil, err
	}
	user, err = updateLastLogin(ctx, tx, user.ID, req.Email, req.EmailVerified)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to commit user auth transaction")
	}
	user.IsNewIdentity = isNewIdentity
	return user, nil
}
