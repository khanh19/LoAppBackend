package lists

import (
	"context"

	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

func requireAuthUserID() (string, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return "", &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}
	return string(userID), nil
}

//encore:api auth method=GET path=/me/library/place-lists
func (s *Service) ListMyPlaceLists(ctx context.Context, params *LibraryQueryParams) (*ListUserPlaceListsResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.listUserPlaceLists(ctx, userID, params)
}

//encore:api auth method=POST path=/me/library/place-lists
func (s *Service) CreateMyPlaceList(ctx context.Context, req *CreateUserPlaceListRequest) (*CreateUserPlaceListResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.createUserPlaceList(ctx, userID, req)
}

//encore:api auth method=GET path=/me/library/place-lists/:id
func (s *Service) GetMyPlaceList(ctx context.Context, id string) (*GetUserPlaceListResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.getUserPlaceList(ctx, userID, id)
}

//encore:api auth method=DELETE path=/me/library/place-lists/:id
func (s *Service) DeleteMyPlaceList(ctx context.Context, id string) error {
	userID, err := requireAuthUserID()
	if err != nil {
		return err
	}
	return s.deleteUserPlaceList(ctx, userID, id)
}

//encore:api auth method=POST path=/me/library/place-lists/:id/places
func (s *Service) AddPlaceToMyPlaceList(ctx context.Context, id string, req *AddPlaceToListRequest) error {
	userID, err := requireAuthUserID()
	if err != nil {
		return err
	}
	return s.addPlaceToUserList(ctx, userID, id, req)
}

//encore:api auth method=DELETE path=/me/library/place-lists/:id/places/:placeID
func (s *Service) RemovePlaceFromMyPlaceList(ctx context.Context, id string, placeID string) error {
	userID, err := requireAuthUserID()
	if err != nil {
		return err
	}
	return s.removePlaceFromUserList(ctx, userID, id, placeID)
}

//encore:api auth method=GET path=/me/library/bookmarks/places
func (s *Service) ListMyPlaceBookmarks(ctx context.Context, params *LibraryQueryParams) (*ListLibraryPlacesResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.listPlaceBookmarks(ctx, userID, params)
}

//encore:api auth method=PUT path=/places/:id/bookmark
func (s *Service) PutPlaceBookmark(ctx context.Context, id string) (*BookmarkStatusResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.putPlaceBookmark(ctx, userID, id)
}

//encore:api auth method=DELETE path=/places/:id/bookmark
func (s *Service) DeletePlaceBookmark(ctx context.Context, id string) (*BookmarkStatusResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.deletePlaceBookmark(ctx, userID, id)
}

//encore:api auth method=GET path=/me/library/stamps
func (s *Service) ListMyLibraryStamps(ctx context.Context, params *LibraryQueryParams) (*ListLibraryPlacesResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.listUserStampedPlaces(ctx, userID, params)
}

//encore:api auth method=GET path=/me/plans
func (s *Service) ListMyPlans(ctx context.Context, params *LibraryQueryParams) (*ListLibraryPlansResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.listMyPlans(ctx, userID, params)
}

//encore:api auth method=GET path=/me/library/bookmarks/plans
func (s *Service) ListMyPlanBookmarks(ctx context.Context, params *LibraryQueryParams) (*ListLibraryPlansResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.listBookmarkedPlans(ctx, userID, params)
}

//encore:api auth method=GET path=/plans/:slug/bookmark
func (s *Service) GetPlanBookmark(ctx context.Context, slug string) (*BookmarkStatusResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.getPlanBookmark(ctx, userID, slug)
}

//encore:api auth method=PUT path=/plans/:slug/bookmark
func (s *Service) PutPlanBookmark(ctx context.Context, slug string) (*BookmarkStatusResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.putPlanBookmark(ctx, userID, slug)
}

//encore:api auth method=DELETE path=/plans/:slug/bookmark
func (s *Service) DeletePlanBookmark(ctx context.Context, slug string) (*BookmarkStatusResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}
	return s.deletePlanBookmark(ctx, userID, slug)
}
