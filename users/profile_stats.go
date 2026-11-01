package users

import (
	"context"

	"encore.app/internal/dbgen"
	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

type UserProfileStats struct {
	FollowersCount   int32 `json:"followers_count"`
	FollowingCount   int32 `json:"following_count"`
	StampsCount      int32 `json:"stamps_count"`
	CollectionsCount int32 `json:"collections_count"`
	PlansCount       int32 `json:"plans_count"`
}

//encore:api auth method=GET path=/users/profile/stats
func (s *Service) GetProfileStats(ctx context.Context) (*UserProfileStats, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}

	uid, err := uuidFromString(string(userID))
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}

	row, err := dbgen.New(s.db).GetUserProfileStats(ctx, uid)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to fetch profile stats")
	}

	return &UserProfileStats{
		FollowersCount:   row.FollowersCount,
		FollowingCount:   row.FollowingCount,
		StampsCount:      row.StampsCount,
		CollectionsCount: row.CollectionsCount,
		PlansCount:       row.PlansCount,
	}, nil
}
