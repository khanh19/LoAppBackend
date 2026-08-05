package stamps

import (
	"context"
	"fmt"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
)

const (
	defaultFeedLimit = 20
	maxFeedLimit     = 50
)

//encore:api auth method=GET path=/feed/stamps
func (s *Service) GetStampFeed(ctx context.Context, p *FeedQuery) (*FeedResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}

	limit := defaultFeedLimit
	offset := 0
	if p != nil {
		limit = normalizeFeedLimit(p.Limit)
		offset = normalizeFeedOffset(p.Offset)
	}

	viewerUUID, err := uuidFromString(userID)
	if err != nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "invalid user"}
	}

	rows, err := dbgen.New(s.db).ListFeedStamps(ctx, dbgen.ListFeedStampsParams{
		ViewerID:  viewerUUID,
		LimitVal:  int32(limit + 1),
		OffsetVal: int32(offset),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list stamp feed")
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	items := make([]FeedStampItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, feedStampItemFromRow(row, userID))
	}

	return &FeedResponse{
		Items:   items,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}

func normalizeFeedLimit(limit int) int {
	if limit <= 0 {
		return defaultFeedLimit
	}
	if limit > maxFeedLimit {
		return maxFeedLimit
	}
	return limit
}

func normalizeFeedOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func feedStampItemFromRow(row dbgen.ListFeedStampsRow, viewerID string) FeedStampItem {
	displayName := strings.TrimSpace(row.FirstName + " " + row.LastName)
	if displayName == "" {
		displayName = row.Username
	}

	var photoURL *string
	if path := strings.TrimSpace(row.PhotoStoragePath); path != "" {
		url := publicStorageURL(path)
		photoURL = &url
	}

	return FeedStampItem{
		StampID:       row.StampID,
		UserID:        row.UserID,
		DisplayName:   displayName,
		Username:      row.Username,
		AvatarURL:     row.AvatarUrl,
		PlaceID:       row.PlaceID,
		PlaceName:     row.PlaceName,
		PlaceImageURL: row.PlaceImageUrl,
		VenueCategory: row.VenueCategory,
		Verdict:       row.Verdict,
		Band:          row.Band,
		PersonalScore: numericToFloatPtr(row.PersonalScore),
		Note:          row.Note,
		PhotoURL:      photoURL,
		IsMine:        row.UserID == viewerID,
		CreatedAt:     timestamptzValue(row.CreatedAt),
	}
}

func publicStorageURL(storagePath string) string {
	base := strings.TrimRight(secrets.SupabaseURL, "/")
	if base == "" {
		base = "https://xofbxjbrzpsocnjdsmmk.supabase.co"
	}
	return fmt.Sprintf("%s/storage/v1/object/public/LoAppBucket/%s", base, strings.TrimLeft(storagePath, "/"))
}
