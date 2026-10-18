package lists

import (
	"context"
	"strings"
	"unicode/utf8"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultLibraryPlansLimit = 20
	maxLibraryPlansLimit     = 50
)

func normalizeLibrarySort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "name":
		return "name"
	case "score":
		return "score"
	default:
		return "recent"
	}
}

func optionalQuery(q string) *string {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	return &q
}

func normalizeLibraryPlansLimit(limit int) int {
	if limit <= 0 {
		return defaultLibraryPlansLimit
	}
	if limit > maxLibraryPlansLimit {
		return maxLibraryPlansLimit
	}
	return limit
}

func normalizeLibraryOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func requireUserUUID(userID string) (pgtype.UUID, error) {
	id, err := uuidFromString(userID)
	if err != nil {
		return pgtype.UUID{}, errs.WrapCode(err, errs.InvalidArgument, "invalid user id")
	}
	return id, nil
}

func (s *Service) listUserPlaceLists(ctx context.Context, userID string, params *LibraryQueryParams) (*ListUserPlaceListsResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	sort := "recent"
	q := ""
	if params != nil {
		sort = normalizeLibrarySort(params.Sort)
		q = params.Q
	}

	dbq := dbgen.New(s.db)
	rows, err := dbq.ListUserPlaceLists(ctx, dbgen.ListUserPlaceListsParams{
		OwnerUserID: ownerID,
		Q:           optionalQuery(q),
		Sort:        sort,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list place lists")
	}

	count, err := dbq.CountPlaceBookmarks(ctx, ownerID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to count place bookmarks")
	}

	lists := make([]UserPlaceListSummary, 0, len(rows))
	for _, row := range rows {
		lists = append(lists, UserPlaceListSummary{
			ID:            row.ID,
			Title:         row.Title,
			ItemCount:     row.ItemCount,
			CoverImageURL: stringPtr(row.CoverImageUrl),
			CreatedAt:     timestamptzToString(row.CreatedAt),
			UpdatedAt:     timestamptzToString(row.UpdatedAt),
		})
	}
	return &ListUserPlaceListsResponse{
		Lists:           lists,
		BookmarkedCount: count,
	}, nil
}

func (s *Service) createUserPlaceList(ctx context.Context, userID string, req *CreateUserPlaceListRequest) (*CreateUserPlaceListResponse, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request body is required"}
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "title is required"}
	}
	if utf8.RuneCountInString(title) > 80 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "title must be 80 characters or fewer"}
	}

	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := dbgen.New(s.db).CreateUserPlaceList(ctx, dbgen.CreateUserPlaceListParams{
		OwnerUserID: ownerID,
		Title:       title,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create place list")
	}

	return &CreateUserPlaceListResponse{
		List: UserPlaceListSummary{
			ID:            row.ID,
			Title:         row.Title,
			ItemCount:     0,
			CoverImageURL: nil,
			CreatedAt:     timestamptzToString(row.CreatedAt),
			UpdatedAt:     timestamptzToString(row.UpdatedAt),
		},
	}, nil
}

func (s *Service) getUserPlaceList(ctx context.Context, userID, listID string) (*GetUserPlaceListResponse, error) {
	if _, err := requireUserUUID(userID); err != nil {
		return nil, err
	}
	listUUID, err := uuidFromString(listID)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid list id"}
	}

	q := dbgen.New(s.db)
	row, err := q.GetUserPlaceListByID(ctx, listUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &errs.Error{Code: errs.NotFound, Message: "list not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to get place list")
	}
	if row.OwnerUserID != userID {
		return nil, &errs.Error{Code: errs.PermissionDenied, Message: "not your list"}
	}

	items, err := q.ListUserPlaceListItems(ctx, listUUID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list place list items")
	}

	out := make([]LibraryPlaceItem, 0, len(items))
	for _, item := range items {
		out = append(out, LibraryPlaceItem{
			PlaceID:       item.PlaceID,
			Name:          item.Name,
			Neighborhood:  item.Neighborhood,
			Address:       item.Address,
			CoverImageURL: libraryCoverURL(item.CoverImageUrl),
			CitySlug:      item.CitySlug,
			CityName:      item.CityName,
			CreatedAt:     timestamptzToString(item.CreatedAt),
		})
	}

	return &GetUserPlaceListResponse{
		List: UserPlaceListSummary{
			ID:            row.ID,
			Title:         row.Title,
			ItemCount:     row.ItemCount,
			CoverImageURL: stringPtr(row.CoverImageUrl),
			CreatedAt:     timestamptzToString(row.CreatedAt),
			UpdatedAt:     timestamptzToString(row.UpdatedAt),
		},
		Items: out,
	}, nil
}

func (s *Service) deleteUserPlaceList(ctx context.Context, userID, listID string) error {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return err
	}
	listUUID, err := uuidFromString(listID)
	if err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid list id"}
	}

	n, err := dbgen.New(s.db).DeleteUserPlaceList(ctx, dbgen.DeleteUserPlaceListParams{
		ID:          listUUID,
		OwnerUserID: ownerID,
	})
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to delete place list")
	}
	if n == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "list not found"}
	}
	return nil
}

func (s *Service) addPlaceToUserList(ctx context.Context, userID, listID string, req *AddPlaceToListRequest) error {
	if req == nil || strings.TrimSpace(req.PlaceID) == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "place_id is required"}
	}
	if _, err := requireUserUUID(userID); err != nil {
		return err
	}
	listUUID, err := uuidFromString(listID)
	if err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid list id"}
	}
	placeUUID, err := uuidFromString(strings.TrimSpace(req.PlaceID))
	if err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid place id"}
	}

	q := dbgen.New(s.db)
	row, err := q.GetUserPlaceListByID(ctx, listUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &errs.Error{Code: errs.NotFound, Message: "list not found"}
		}
		return errs.WrapCode(err, errs.Internal, "failed to get place list")
	}
	if row.OwnerUserID != userID {
		return &errs.Error{Code: errs.PermissionDenied, Message: "not your list"}
	}

	if _, err := q.PlaceExists(ctx, placeUUID); err != nil {
		if err == pgx.ErrNoRows {
			return &errs.Error{Code: errs.NotFound, Message: "place not found"}
		}
		return errs.WrapCode(err, errs.Internal, "failed to verify place")
	}

	if err := q.AddUserPlaceListItem(ctx, dbgen.AddUserPlaceListItemParams{
		ListID:  listUUID,
		PlaceID: placeUUID,
	}); err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to add place to list")
	}
	_ = q.TouchUserPlaceList(ctx, listUUID)
	return nil
}

func (s *Service) removePlaceFromUserList(ctx context.Context, userID, listID, placeID string) error {
	listUUID, err := uuidFromString(listID)
	if err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid list id"}
	}
	placeUUID, err := uuidFromString(placeID)
	if err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid place id"}
	}

	q := dbgen.New(s.db)
	row, err := q.GetUserPlaceListByID(ctx, listUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &errs.Error{Code: errs.NotFound, Message: "list not found"}
		}
		return errs.WrapCode(err, errs.Internal, "failed to get place list")
	}
	if row.OwnerUserID != userID {
		return &errs.Error{Code: errs.PermissionDenied, Message: "not your list"}
	}

	n, err := q.RemoveUserPlaceListItem(ctx, dbgen.RemoveUserPlaceListItemParams{
		ListID:  listUUID,
		PlaceID: placeUUID,
	})
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to remove place from list")
	}
	if n == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "place not in list"}
	}
	_ = q.TouchUserPlaceList(ctx, listUUID)
	return nil
}

func (s *Service) listPlaceBookmarks(ctx context.Context, userID string, params *LibraryQueryParams) (*ListLibraryPlacesResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	sort := "recent"
	q := ""
	if params != nil {
		sort = normalizeLibrarySort(params.Sort)
		q = params.Q
	}

	rows, err := dbgen.New(s.db).ListPlaceBookmarks(ctx, dbgen.ListPlaceBookmarksParams{
		UserID: ownerID,
		Q:      optionalQuery(q),
		Sort:   sort,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list place bookmarks")
	}

	places := make([]LibraryPlaceItem, 0, len(rows))
	for _, row := range rows {
		places = append(places, LibraryPlaceItem{
			PlaceID:       row.PlaceID,
			Name:          row.Name,
			Neighborhood:  row.Neighborhood,
			Address:       row.Address,
			CoverImageURL: libraryCoverURL(row.CoverImageUrl),
			PriceLevel:    row.PriceLevel,
			Tags:          row.Tags,
			Rating:        numericPtr(row.RatingCached),
			CitySlug:      row.CitySlug,
			CityName:      row.CityName,
			CreatedAt:     timestamptzToString(row.CreatedAt),
		})
	}
	return &ListLibraryPlacesResponse{Places: places}, nil
}

func (s *Service) putPlaceBookmark(ctx context.Context, userID, placeID string) (*BookmarkStatusResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	placeUUID, err := uuidFromString(placeID)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place id"}
	}

	q := dbgen.New(s.db)
	if _, err := q.PlaceExists(ctx, placeUUID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, &errs.Error{Code: errs.NotFound, Message: "place not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to verify place")
	}

	if err := q.UpsertPlaceBookmark(ctx, dbgen.UpsertPlaceBookmarkParams{
		UserID:  ownerID,
		PlaceID: placeUUID,
	}); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to bookmark place")
	}
	return &BookmarkStatusResponse{Bookmarked: true}, nil
}

func (s *Service) deletePlaceBookmark(ctx context.Context, userID, placeID string) (*BookmarkStatusResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	placeUUID, err := uuidFromString(placeID)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid place id"}
	}

	_, err = dbgen.New(s.db).DeletePlaceBookmark(ctx, dbgen.DeletePlaceBookmarkParams{
		UserID:  ownerID,
		PlaceID: placeUUID,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to remove place bookmark")
	}
	return &BookmarkStatusResponse{Bookmarked: false}, nil
}

func (s *Service) listUserStampedPlaces(ctx context.Context, userID string, params *LibraryQueryParams) (*ListLibraryPlacesResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	sort := "recent"
	q := ""
	if params != nil {
		sort = normalizeLibrarySort(params.Sort)
		q = params.Q
	}

	rows, err := dbgen.New(s.db).ListUserStampedPlaces(ctx, dbgen.ListUserStampedPlacesParams{
		UserID: ownerID,
		Q:      optionalQuery(q),
		Sort:   sort,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list stamped places")
	}

	places := make([]LibraryPlaceItem, 0, len(rows))
	for _, row := range rows {
		places = append(places, LibraryPlaceItem{
			PlaceID:       row.PlaceID,
			Name:          row.Name,
			Neighborhood:  row.Neighborhood,
			Address:       row.Address,
			CoverImageURL: libraryCoverURL(row.CoverImageUrl),
			PriceLevel:    row.PriceLevel,
			Tags:          row.Tags,
			StampScore:    numericPtr(row.StampScore),
			CitySlug:      row.CitySlug,
			CityName:      row.CityName,
			CreatedAt:     timestamptzToString(row.StampedAt),
		})
	}
	return &ListLibraryPlacesResponse{Places: places}, nil
}

func (s *Service) listMyPlans(ctx context.Context, userID string, params *LibraryQueryParams) (*ListLibraryPlansResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	limit, offset, sort, q := parseLibraryPlanPaging(params)

	dbq := dbgen.New(s.db)
	total, err := dbq.CountMyPlans(ctx, dbgen.CountMyPlansParams{
		CreatorUserID: ownerID,
		Q:             optionalQuery(q),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to count my plans")
	}

	rows, err := dbq.ListMyPlans(ctx, dbgen.ListMyPlansParams{
		CreatorUserID: ownerID,
		Q:             optionalQuery(q),
		Sort:          sort,
		OffsetVal:     int32(offset),
		LimitVal:      int32(limit),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list my plans")
	}

	plans := make([]LibraryPlanSummary, 0, len(rows))
	for _, row := range rows {
		plans = append(plans, LibraryPlanSummary{
			ID:                 row.ID,
			Slug:               row.Slug,
			Title:              row.Title,
			Subtitle:           row.Subtitle,
			Category:           row.Category,
			Area:               row.Area,
			CitySlug:           row.CitySlug,
			CityName:           row.CityName,
			SavesCount:         row.SavesCount,
			StopsCount:         row.StopsCount,
			Visibility:         row.Visibility,
			CoverPlaceID:       stringPtr(row.CoverPlaceID),
			CoverImageURL:      stringPtr(row.CoverImageUrl),
			CreatorDisplayName: row.CreatorDisplayName,
			CreatorUsername:    row.CreatorUsername,
			CreatorAvatarURL:   row.CreatorAvatarUrl,
			CreatedAt:          timestamptzToString(row.CreatedAt),
		})
	}
	return &ListLibraryPlansResponse{
		Plans:  plans,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) listBookmarkedPlans(ctx context.Context, userID string, params *LibraryQueryParams) (*ListLibraryPlansResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	limit, offset, sort, q := parseLibraryPlanPaging(params)

	dbq := dbgen.New(s.db)
	total, err := dbq.CountBookmarkedPlans(ctx, dbgen.CountBookmarkedPlansParams{
		UserID: ownerID,
		Q:      optionalQuery(q),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to count bookmarked plans")
	}

	rows, err := dbq.ListBookmarkedPlans(ctx, dbgen.ListBookmarkedPlansParams{
		UserID:    ownerID,
		Q:         optionalQuery(q),
		Sort:      sort,
		OffsetVal: int32(offset),
		LimitVal:  int32(limit),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list bookmarked plans")
	}

	plans := make([]LibraryPlanSummary, 0, len(rows))
	for _, row := range rows {
		bookmarkedAt := timestamptzToString(row.BookmarkedAt)
		plans = append(plans, LibraryPlanSummary{
			ID:                 row.ID,
			Slug:               row.Slug,
			Title:              row.Title,
			Subtitle:           row.Subtitle,
			Category:           row.Category,
			Area:               row.Area,
			CitySlug:           row.CitySlug,
			CityName:           row.CityName,
			SavesCount:         row.SavesCount,
			StopsCount:         row.StopsCount,
			Visibility:         row.Visibility,
			CoverPlaceID:       stringPtr(row.CoverPlaceID),
			CoverImageURL:      stringPtr(row.CoverImageUrl),
			CreatorDisplayName: row.CreatorDisplayName,
			CreatorUsername:    row.CreatorUsername,
			CreatorAvatarURL:   row.CreatorAvatarUrl,
			CreatedAt:          timestamptzToString(row.CreatedAt),
			BookmarkedAt:       &bookmarkedAt,
		})
	}
	return &ListLibraryPlansResponse{
		Plans:  plans,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func parseLibraryPlanPaging(params *LibraryQueryParams) (limit, offset int, sort, q string) {
	limit = defaultLibraryPlansLimit
	sort = "recent"
	if params != nil {
		limit = normalizeLibraryPlansLimit(params.Limit)
		offset = normalizeLibraryOffset(params.Offset)
		sort = normalizeLibrarySort(params.Sort)
		q = params.Q
	}
	return limit, offset, sort, q
}

func (s *Service) putPlanBookmark(ctx context.Context, userID, slug string) (*BookmarkStatusResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "slug is required"}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to begin bookmark transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	plan, err := q.GetPlanIDBySlug(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &errs.Error{Code: errs.NotFound, Message: "plan not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to resolve plan")
	}
	listUUID, err := uuidFromString(plan.ID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid plan id")
	}

	exists, err := q.HasPlanBookmark(ctx, dbgen.HasPlanBookmarkParams{
		UserID: ownerID,
		ListID: listUUID,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to check plan bookmark")
	}
	if !exists {
		if err := q.UpsertPlanBookmark(ctx, dbgen.UpsertPlanBookmarkParams{
			UserID: ownerID,
			ListID: listUUID,
		}); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to bookmark plan")
		}
		if err := q.IncrementPlanSavesCount(ctx, listUUID); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to update saves count")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to commit bookmark")
	}
	return &BookmarkStatusResponse{Bookmarked: true}, nil
}

func (s *Service) deletePlanBookmark(ctx context.Context, userID, slug string) (*BookmarkStatusResponse, error) {
	ownerID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "slug is required"}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to begin unbookmark transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	plan, err := q.GetPlanIDBySlug(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &errs.Error{Code: errs.NotFound, Message: "plan not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to resolve plan")
	}
	listUUID, err := uuidFromString(plan.ID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "invalid plan id")
	}

	n, err := q.DeletePlanBookmark(ctx, dbgen.DeletePlanBookmarkParams{
		UserID: ownerID,
		ListID: listUUID,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to remove plan bookmark")
	}
	if n > 0 {
		if err := q.DecrementPlanSavesCount(ctx, listUUID); err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to update saves count")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to commit unbookmark")
	}
	return &BookmarkStatusResponse{Bookmarked: false}, nil
}
