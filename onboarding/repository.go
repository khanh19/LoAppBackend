package onboarding

import (
	"context"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listCities(ctx context.Context, db *pgxpool.Pool) ([]City, error) {
	rows, err := dbgen.New(db).ListActiveCities(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list cities")
	}

	cities := make([]City, 0, len(rows))
	for _, row := range rows {
		cities = append(cities, City{
			ID:            row.ID,
			Slug:          row.Slug,
			Name:          row.Name,
			CountryCode:   row.CountryCode,
			CoverImageURL: row.CoverImageUrl,
		})
	}
	return cities, nil
}

func listCategories(ctx context.Context, db *pgxpool.Pool) ([]PurposeCategory, error) {
	rows, err := dbgen.New(db).ListActivePurposeCategories(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list categories")
	}

	categories := make([]PurposeCategory, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, PurposeCategory{
			ID:       row.ID,
			Slug:     row.Slug,
			Name:     row.Name,
			IconName: row.IconName,
		})
	}
	return categories, nil
}

func listVibes(ctx context.Context, db *pgxpool.Pool) ([]Vibe, error) {
	rows, err := dbgen.New(db).ListActiveVibes(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list vibes")
	}

	vibes := make([]Vibe, 0, len(rows))
	for _, row := range rows {
		vibes = append(vibes, Vibe{
			ID:          row.ID,
			Slug:        row.Slug,
			Name:        row.Name,
			Description: row.Description,
		})
	}
	return vibes, nil
}

func listVenuesByCity(ctx context.Context, db *pgxpool.Pool, cityID string) ([]Venue, error) {
	cid, err := uuidFromString(cityID)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "city_id is invalid"}
	}

	rows, err := dbgen.New(db).ListCuratedVenuesByCity(ctx, cid)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list venues")
	}

	venues := make([]Venue, 0, len(rows))
	for _, row := range rows {
		venues = append(venues, Venue{
			ID:       row.ID,
			Slug:     row.Slug,
			Name:     row.Name,
			Category: row.Category,
			Rating:   numericToFloat(row.RatingCached),
			Price:    formatPriceLevel(row.PriceLevel),
			Location: stringValue(row.Neighborhood),
			Tags:     row.Tags,
			Image:    row.CoverImageUrl,
		})
	}
	return venues, nil
}

func completeOnboarding(ctx context.Context, db *pgxpool.Pool, userID string, req *CompleteOnboardingRequest) (string, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return "", errs.WrapCode(err, errs.Internal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	q := dbgen.New(tx)
	uid, err := uuidFromString(userID)
	if err != nil {
		return "", &errs.Error{Code: errs.InvalidArgument, Message: "invalid user id"}
	}

	if !req.Skip {
		if err := saveOnboardingPreferences(ctx, q, uid, req); err != nil {
			return "", err
		}
	}

	status, err := q.CompleteUserOnboarding(ctx, uid)
	if err != nil {
		return "", errs.WrapCode(err, errs.Internal, "failed to complete onboarding")
	}

	if err := tx.Commit(ctx); err != nil {
		return "", errs.WrapCode(err, errs.Internal, "failed to commit onboarding completion")
	}
	return status, nil
}

func saveOnboardingPreferences(ctx context.Context, q *dbgen.Queries, uid pgtype.UUID, req *CompleteOnboardingRequest) error {
	if err := q.DeleteUserExploreCities(ctx, uid); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to reset explore cities")
	}
	for i, cityID := range req.CityIDs {
		cid, err := uuidFromString(cityID)
		if err != nil {
			return &errs.Error{Code: errs.InvalidArgument, Message: "cityIds contains an invalid id"}
		}
		if err := q.InsertUserExploreCity(ctx, dbgen.InsertUserExploreCityParams{
			UserID:   uid,
			CityID:   cid,
			Priority: int32(i + 1),
		}); err != nil {
			return errs.WrapCode(err, errs.Internal, "failed to save explore city")
		}
	}

	if err := q.DeleteUserPurposeCategories(ctx, uid); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to reset purpose categories")
	}
	for _, categoryID := range req.CategoryIDs {
		cid, err := uuidFromString(categoryID)
		if err != nil {
			return &errs.Error{Code: errs.InvalidArgument, Message: "categoryIds contains an invalid id"}
		}
		if err := q.InsertUserPurposeCategory(ctx, dbgen.InsertUserPurposeCategoryParams{
			UserID:     uid,
			CategoryID: cid,
		}); err != nil {
			return errs.WrapCode(err, errs.Internal, "failed to save purpose category")
		}
	}

	if err := q.DeleteUserVibes(ctx, uid); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to reset vibes")
	}
	for _, vibeID := range req.VibeIDs {
		vid, err := uuidFromString(vibeID)
		if err != nil {
			return &errs.Error{Code: errs.InvalidArgument, Message: "vibeIds contains an invalid id"}
		}
		if err := q.InsertUserVibe(ctx, dbgen.InsertUserVibeParams{
			UserID: uid,
			VibeID: vid,
			Weight: numericFromFloat(1),
		}); err != nil {
			return errs.WrapCode(err, errs.Internal, "failed to save vibe")
		}
	}

	if err := q.DeleteUserLikedPlacesBySource(ctx, dbgen.DeleteUserLikedPlacesBySourceParams{
		UserID: uid,
		Source: "onboarding",
	}); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to reset liked places")
	}
	for _, placeID := range req.PlaceIDs {
		pid, err := uuidFromString(placeID)
		if err != nil {
			return &errs.Error{Code: errs.InvalidArgument, Message: "placeIds contains an invalid id"}
		}
		if err := q.InsertUserLikedPlace(ctx, dbgen.InsertUserLikedPlaceParams{
			UserID:  uid,
			PlaceID: pid,
			Source:  "onboarding",
		}); err != nil {
			return errs.WrapCode(err, errs.Internal, "failed to save liked place")
		}
	}

	return nil
}

func formatPriceLevel(level *int16) string {
	if level == nil {
		return ""
	}
	switch *level {
	case 1:
		return "$5–$10"
	case 2:
		return "$10–$20"
	case 3:
		return "$20–$30"
	case 4:
		return "$30+"
	default:
		return ""
	}
}
