package lists

import (
	"context"
	"errors"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
)

func (s *Service) updatePlan(ctx context.Context, userID, slug string, req *UpdatePlanRequest) (*PlaceListDetail, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "slug is required"}
	}

	normalized, err := normalizeUpdatePlanRequest(req)
	if err != nil {
		return nil, err
	}

	creatorID, err := uuidFromString(userID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "invalid user id")
	}

	existing, err := dbgen.New(s.db).GetPlaceListBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "plan not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to load plan")
	}
	if existing.ListType != "user_plan" {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "only user plans can be updated"}
	}
	if existing.CreatorUserID != userID {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "only the plan creator can update this plan"}
	}

	listUUID, err := uuidFromString(existing.ID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid plan id")
	}
	cityID, err := uuidFromString(existing.CityID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid city id")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to begin plan update transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := dbgen.New(tx)
	n, err := q.UpdateUserPlanTitle(ctx, dbgen.UpdateUserPlanTitleParams{
		Title:         normalized.Title,
		ID:            listUUID,
		CreatorUserID: creatorID,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update plan title")
	}
	if n == 0 {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "only the plan creator can update this plan"}
	}

	if err := q.DeletePlaceListEntriesByListID(ctx, listUUID); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to clear plan stops")
	}

	for _, stop := range normalized.Stops {
		placeID, err := s.createOrUsePlanStopPlace(ctx, q, cityID, existing.CitySlug, stop)
		if err != nil {
			return nil, err
		}
		placeUUID, err := uuidFromString(placeID)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "invalid stop place id")
		}
		if err := q.CreatePlanEntry(ctx, dbgen.CreatePlanEntryParams{
			ListID:       listUUID,
			PlaceID:      placeUUID,
			SeedName:     stop.PlaceName,
			StopOrder:    stop.StopOrder,
			Note:         stop.Note,
			TimeLabel:    stop.TimeLabel,
			ActivityType: stop.ActivityType,
			ImageUrl:     stop.ImageURL,
		}); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to create plan stop")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to commit plan update")
	}

	return getPlaceListBySlug(ctx, s.db, slug)
}

func (s *Service) deletePlan(ctx context.Context, userID, slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "slug is required"}
	}

	creatorID, err := uuidFromString(userID)
	if err != nil {
		return errs.WrapCode(err, errs.InvalidArgument, "invalid user id")
	}

	existing, err := dbgen.New(s.db).GetPlaceListBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &errs.Error{Code: errs.NotFound, Message: "plan not found"}
		}
		return errs.WrapCode(err, errs.Internal, "failed to load plan")
	}
	if existing.ListType != "user_plan" {
		return &errs.Error{Code: errs.PermissionDenied, Message: "only user plans can be deleted"}
	}
	if existing.CreatorUserID != userID {
		return &errs.Error{Code: errs.PermissionDenied, Message: "only the plan creator can delete this plan"}
	}

	listUUID, err := uuidFromString(existing.ID)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "invalid plan id")
	}

	n, err := dbgen.New(s.db).DeactivateUserPlan(ctx, dbgen.DeactivateUserPlanParams{
		ID:            listUUID,
		CreatorUserID: creatorID,
	})
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to delete plan")
	}
	if n == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "plan not found"}
	}
	return nil
}

func normalizeUpdatePlanRequest(req *UpdatePlanRequest) (*UpdatePlanRequest, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}
	normalized := *req
	normalized.Title = strings.TrimSpace(normalized.Title)
	if normalized.Title == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "title is required"}
	}
	if len(normalized.Stops) == 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "at least one stop is required"}
	}

	normalized.Stops = append([]CreatePlanStopRequest(nil), normalized.Stops...)
	seenStopOrders := make(map[int32]struct{}, len(normalized.Stops))
	seenPlaceNames := make(map[string]struct{}, len(normalized.Stops))
	for i := range normalized.Stops {
		stop, err := normalizeCreatePlanStop(normalized.Stops[i], i+1)
		if err != nil {
			return nil, err
		}
		if _, ok := seenStopOrders[stop.StopOrder]; ok {
			return nil, &errs.Error{Code: errs.InvalidArgument, Message: "stop_order values must be unique"}
		}
		seenStopOrders[stop.StopOrder] = struct{}{}
		placeNameKey := strings.ToLower(stop.PlaceName)
		if _, ok := seenPlaceNames[placeNameKey]; ok {
			return nil, &errs.Error{Code: errs.InvalidArgument, Message: "stop place_name values must be unique"}
		}
		seenPlaceNames[placeNameKey] = struct{}{}
		normalized.Stops[i] = stop
	}
	return &normalized, nil
}
