package users

import (
	"context"
	"errors"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
)

func findByIdentity(ctx context.Context, tx pgx.Tx, provider, providerUserID string) (*User, error) {
	q := dbgen.New(tx)
	row, err := q.FindUserByIdentity(ctx, dbgen.FindUserByIdentityParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to query user identity")
	}
	return userFromIdentityRow(row), nil
}

func findByEmail(ctx context.Context, tx pgx.Tx, email string) (*User, error) {
	q := dbgen.New(tx)
	row, err := q.FindUserByEmail(ctx, &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to query user by email")
	}
	return userFromEmailRow(row), nil
}

func createUser(ctx context.Context, tx pgx.Tx, email *string, emailVerified bool) (*User, error) {
	q := dbgen.New(tx)
	row, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		PrimaryEmail:  email,
		EmailVerified: emailVerified,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create user")
	}
	return userFromCreateRow(row), nil
}

func upsertIdentity(ctx context.Context, tx pgx.Tx, userID string, req *UpsertFromAuthParams, rawProfile []byte) error {
	q := dbgen.New(tx)
	uuid, err := uuidFromString(userID)
	if err != nil {
		return errs.WrapCode(err, errs.InvalidArgument, "invalid user id")
	}

	err = q.UpsertIdentity(ctx, dbgen.UpsertIdentityParams{
		UserID:         uuid,
		Provider:       req.Provider,
		ProviderUserID: req.ProviderUserID,
		Email:          req.Email,
		EmailVerified:  req.EmailVerified,
		RawProfile:     rawProfile,
	})
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to upsert auth identity")
	}
	return nil
}

func updateLastLogin(ctx context.Context, tx pgx.Tx, userID string, email *string, emailVerified bool) (*User, error) {
	q := dbgen.New(tx)
	uuid, err := uuidFromString(userID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "invalid user id")
	}

	row, err := q.UpdateLastLogin(ctx, dbgen.UpdateLastLoginParams{
		UserID:        uuid,
		EmailVerified: emailVerified,
		PrimaryEmail:  email,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update last login")
	}
	return userFromUpdateRow(row), nil
}

func hasUserProfile(ctx context.Context, tx pgx.Tx, userID string) (bool, error) {
	q := dbgen.New(tx)
	uuid, err := uuidFromString(userID)
	if err != nil {
		return false, errs.WrapCode(err, errs.InvalidArgument, "invalid user id")
	}
	return q.HasUserProfile(ctx, uuid)
}

func userFromIdentityRow(row dbgen.FindUserByIdentityRow) *User {
	return &User{
		ID:              row.ID,
		PrimaryEmail:    row.PrimaryEmail,
		EmailVerifiedAt: timestamptzPtr(row.EmailVerifiedAt),
		OnboardingState: row.OnboardingStatus,
		LastLoginAt:     timestamptzPtr(row.LastLoginAt),
		CreatedAt:       timestamptzValue(row.CreatedAt),
		UpdatedAt:       timestamptzValue(row.UpdatedAt),
	}
}

func userFromEmailRow(row dbgen.FindUserByEmailRow) *User {
	return &User{
		ID:              row.ID,
		PrimaryEmail:    row.PrimaryEmail,
		EmailVerifiedAt: timestamptzPtr(row.EmailVerifiedAt),
		OnboardingState: row.OnboardingStatus,
		LastLoginAt:     timestamptzPtr(row.LastLoginAt),
		CreatedAt:       timestamptzValue(row.CreatedAt),
		UpdatedAt:       timestamptzValue(row.UpdatedAt),
	}
}

func userFromCreateRow(row dbgen.CreateUserRow) *User {
	return &User{
		ID:              row.ID,
		PrimaryEmail:    row.PrimaryEmail,
		EmailVerifiedAt: timestamptzPtr(row.EmailVerifiedAt),
		OnboardingState: row.OnboardingStatus,
		LastLoginAt:     timestamptzPtr(row.LastLoginAt),
		CreatedAt:       timestamptzValue(row.CreatedAt),
		UpdatedAt:       timestamptzValue(row.UpdatedAt),
	}
}

func userFromUpdateRow(row dbgen.UpdateLastLoginRow) *User {
	return &User{
		ID:              row.ID,
		PrimaryEmail:    row.PrimaryEmail,
		EmailVerifiedAt: timestamptzPtr(row.EmailVerifiedAt),
		OnboardingState: row.OnboardingStatus,
		LastLoginAt:     timestamptzPtr(row.LastLoginAt),
		CreatedAt:       timestamptzValue(row.CreatedAt),
		UpdatedAt:       timestamptzValue(row.UpdatedAt),
	}
}
