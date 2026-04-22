package users

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"encore.app/pkg/externaldb"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//encore:service
type Service struct {
	db *pgxpool.Pool
}

func initService() (*Service, error) {
	db, err := externaldb.Get(context.Background())
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to initialize users database")
	}
	return &Service{db: db}, nil
}

type User struct {
	ID              string     `json:"id"`
	PrimaryEmail    *string    `json:"primary_email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	OnboardingState string     `json:"onboarding_status"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UpsertFromAuthParams struct {
	Provider       string                 `json:"provider"`
	ProviderUserID string                 `json:"provider_user_id"`
	Email          *string                `json:"email"`
	EmailVerified  bool                   `json:"email_verified"`
	RawProfile     json.RawMessage        `json:"raw_profile"`
}

//encore:api private method=POST path=/users/upsert-from-auth
func (s *Service) UpsertFromAuth(ctx context.Context, req *UpsertFromAuthParams) (*User, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	req.Provider = strings.TrimSpace(req.Provider)
	req.ProviderUserID = strings.TrimSpace(req.ProviderUserID)
	if req.Provider == "" || req.ProviderUserID == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "provider and provider_user_id are required"}
	}
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if email == "" {
			req.Email = nil
		} else {
			req.Email = &email
		}
	}

	rawProfile := req.RawProfile
	if len(rawProfile) == 0 {
		rawProfile = json.RawMessage(`{}`)
	}
	if !json.Valid(rawProfile) {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "raw_profile must be valid JSON"}
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	user, err := findByIdentity(ctx, tx, req.Provider, req.ProviderUserID)
	if err != nil {
		return nil, err
	}
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
	return user, nil
}

func findByIdentity(ctx context.Context, tx pgx.Tx, provider, providerUserID string) (*User, error) {
	row := tx.QueryRow(ctx, `
		SELECT u.id::text, u.primary_email, u.email_verified_at, u.onboarding_status::text, u.last_login_at, u.created_at, u.updated_at
		FROM user_auth_identities i
		JOIN users u ON u.id = i.user_id
		WHERE i.provider = $1::auth_provider
			AND i.provider_user_id = $2
			AND u.deleted_at IS NULL
	`, provider, providerUserID)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to query user identity")
	}
	return user, nil
}

func findByEmail(ctx context.Context, tx pgx.Tx, email string) (*User, error) {
	row := tx.QueryRow(ctx, `
		SELECT id::text, primary_email, email_verified_at, onboarding_status::text, last_login_at, created_at, updated_at
		FROM users
		WHERE primary_email = $1
			AND deleted_at IS NULL
	`, email)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to query user by email")
	}
	return user, nil
}

func createUser(ctx context.Context, tx pgx.Tx, email *string, emailVerified bool) (*User, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO users (primary_email, email_verified_at)
		VALUES ($1, CASE WHEN $2 THEN now() ELSE NULL END)
		RETURNING id::text, primary_email, email_verified_at, onboarding_status::text, last_login_at, created_at, updated_at
	`, email, emailVerified)
	user, err := scanUser(row)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create user")
	}
	return user, nil
}

func upsertIdentity(ctx context.Context, tx pgx.Tx, userID string, req *UpsertFromAuthParams, rawProfile []byte) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_auth_identities (
			user_id, provider, provider_user_id, email, email_verified, raw_profile, linked_at, last_used_at
		)
		VALUES ($1::uuid, $2::auth_provider, $3, $4, $5, $6::jsonb, now(), now())
		ON CONFLICT (provider, provider_user_id)
		DO UPDATE SET
			user_id = EXCLUDED.user_id,
			email = EXCLUDED.email,
			email_verified = EXCLUDED.email_verified,
			raw_profile = EXCLUDED.raw_profile,
			last_used_at = now()
	`, userID, req.Provider, req.ProviderUserID, req.Email, req.EmailVerified, rawProfile)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to upsert auth identity")
	}
	return nil
}

func updateLastLogin(ctx context.Context, tx pgx.Tx, userID string, email *string, emailVerified bool) (*User, error) {
	row := tx.QueryRow(ctx, `
		UPDATE users
		SET
			last_login_at = now(),
			email_verified_at = CASE
				WHEN $2::boolean AND email_verified_at IS NULL THEN now()
				ELSE email_verified_at
			END,
			primary_email = COALESCE(primary_email, $3)
		WHERE id = $1::uuid
		RETURNING id::text, primary_email, email_verified_at, onboarding_status::text, last_login_at, created_at, updated_at
	`, userID, emailVerified, email)
	user, err := scanUser(row)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update last login")
	}
	return user, nil
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID,
		&u.PrimaryEmail,
		&u.EmailVerifiedAt,
		&u.OnboardingState,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
