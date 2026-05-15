package users

import (
	"context"
	"errors"

	"encore.app/internal/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errUserProfileNotFound = errors.New("user profile not found")

func upsertUserProfile(ctx context.Context, db *pgxpool.Pool, userID string, req *UpsertUserProfileRequest) (*UserProfile, error) {
	q := dbgen.New(db)
	uid, err := uuidFromString(userID)
	if err != nil {
		return nil, err
	}

	row, err := q.UpsertUserProfile(ctx, dbgen.UpsertUserProfileParams{
		UserID:              uid,
		FirstName:           req.FirstName,
		LastName:            req.LastName,
		Username:            req.Username,
		PhoneE164:           req.PhoneE164,
		PhoneCountryCode:    req.PhoneCountryCode,
		PhoneNationalNumber: req.PhoneNationalNumber,
		DateOfBirth:         dateFromString(req.DateOfBirth),
		AvatarObjectKey:     req.AvatarObjectKey,
		AvatarUrl:           req.AvatarURL,
		Bio:                 req.Bio,
	})
	if err != nil {
		return nil, err
	}
	return userProfileFromUpsertRow(row), nil
}

func getUserProfile(ctx context.Context, db *pgxpool.Pool, userID string) (*UserProfile, error) {
	q := dbgen.New(db)
	uid, err := uuidFromString(userID)
	if err != nil {
		return nil, err
	}

	row, err := q.GetUserProfile(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errUserProfileNotFound
		}
		return nil, err
	}
	return userProfileFromGetRow(row), nil
}

func isUsernameAvailable(ctx context.Context, db *pgxpool.Pool, username, currentUserID string) (bool, error) {
	q := dbgen.New(db)
	uid, err := uuidFromString(currentUserID)
	if err != nil {
		return false, err
	}
	taken, err := q.IsUsernameTakenByAnotherUser(ctx, dbgen.IsUsernameTakenByAnotherUserParams{
		Username:      username,
		CurrentUserID: uid,
	})
	if err != nil {
		return false, err
	}
	return !taken, nil
}

func userProfileFromUpsertRow(row dbgen.UpsertUserProfileRow) *UserProfile {
	return &UserProfile{
		UserID:              row.UserID,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Username:            row.Username,
		PhoneE164:           row.PhoneE164,
		PhoneCountryCode:    row.PhoneCountryCode,
		PhoneNationalNumber: row.PhoneNationalNumber,
		DateOfBirth:         dateToStringPtr(row.DateOfBirth),
		AvatarObjectKey:     row.AvatarObjectKey,
		AvatarURL:           row.AvatarUrl,
		Bio:                 row.Bio,
		CreatedAt:           timestamptzValue(row.CreatedAt),
		UpdatedAt:           timestamptzValue(row.UpdatedAt),
	}
}

func userProfileFromGetRow(row dbgen.GetUserProfileRow) *UserProfile {
	return &UserProfile{
		UserID:              row.UserID,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Username:            row.Username,
		PhoneE164:           row.PhoneE164,
		PhoneCountryCode:    row.PhoneCountryCode,
		PhoneNationalNumber: row.PhoneNationalNumber,
		DateOfBirth:         dateToStringPtr(row.DateOfBirth),
		AvatarObjectKey:     row.AvatarObjectKey,
		AvatarURL:           row.AvatarUrl,
		Bio:                 row.Bio,
		CreatedAt:           timestamptzValue(row.CreatedAt),
		UpdatedAt:           timestamptzValue(row.UpdatedAt),
	}
}
