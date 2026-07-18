package lists

import (
	"context"
	"strings"

	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

//encore:api public method=GET path=/lists
func (s *Service) ListPlaceLists(ctx context.Context) (*ListPlaceListsResponse, error) {
	lists, err := listPlaceLists(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return &ListPlaceListsResponse{Lists: lists}, nil
}

//encore:api private method=GET path=/lists/sync-runs
func (s *Service) ListSyncRuns(ctx context.Context, params *ListSyncRunsParams) (*ListSyncRunsResponse, error) {
	limit := 20
	if params != nil && params.Limit > 0 {
		limit = params.Limit
	}
	runs, err := listRecentSyncRuns(ctx, s.db, limit)
	if err != nil {
		return nil, err
	}
	return &ListSyncRunsResponse{Runs: runs}, nil
}

//encore:api private method=POST path=/lists/sync
func (s *Service) SyncPlaceLists(ctx context.Context) (*SyncPlaceListsResponse, error) {
	return s.syncPlaceLists(ctx)
}

//encore:api public method=GET path=/lists/details/:slug
func (s *Service) GetPlaceList(ctx context.Context, slug string) (*GetPlaceListResponse, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "slug is required"}
	}
	list, err := getPlaceListBySlug(ctx, s.db, slug)
	if err != nil {
		return nil, err
	}
	return &GetPlaceListResponse{List: *list}, nil
}

//encore:api public method=GET path=/plans
func (s *Service) ListPlans(ctx context.Context, params *ListPlansParams) (*ListPlansResponse, error) {
	limit := defaultPlansLimit
	citySlug := ""
	if params != nil {
		limit = normalizePlansLimit(params.Limit)
		citySlug = params.CitySlug
	}

	plans, err := listPlans(ctx, s.db, citySlug, limit)
	if err != nil {
		return nil, err
	}
	return &ListPlansResponse{Plans: plans}, nil
}

//encore:api auth method=POST path=/plans
func (s *Service) CreatePlan(ctx context.Context, req *CreatePlanRequest) (*CreatePlanResponse, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}
	plan, err := s.createPlan(ctx, string(userID), req)
	if err != nil {
		return nil, err
	}
	return &CreatePlanResponse{Plan: *plan}, nil
}
