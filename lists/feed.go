package lists

import (
	"context"
	"strings"

	"encore.app/internal/dbgen"
	"encore.dev/beta/errs"
)

const (
	defaultFeedLimit = 20
	maxFeedLimit     = 50
)

//encore:api auth method=GET path=/feed/plans
func (s *Service) GetPlanFeed(ctx context.Context, p *FeedQuery) (*FeedPlansResponse, error) {
	userID, err := requireAuthUserID()
	if err != nil {
		return nil, err
	}

	limit := defaultFeedLimit
	offset := 0
	if p != nil {
		limit = normalizeFeedLimit(p.Limit)
		offset = normalizeFeedOffset(p.Offset)
	}

	viewerUUID, err := requireUserUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := dbgen.New(s.db).ListFeedPlans(ctx, dbgen.ListFeedPlansParams{
		ViewerID:  viewerUUID,
		LimitVal:  int32(limit + 1),
		OffsetVal: int32(offset),
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to list plan feed")
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	items := make([]FeedPlanItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, feedPlanItemFromRow(row, userID))
	}

	return &FeedPlansResponse{
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

func feedPlanItemFromRow(row dbgen.ListFeedPlansRow, viewerID string) FeedPlanItem {
	displayName := ""
	if row.CreatorDisplayName != nil {
		displayName = strings.TrimSpace(*row.CreatorDisplayName)
	}
	if displayName == "" {
		displayName = row.CreatorUsername
	}

	return FeedPlanItem{
		PlanID:             row.PlanID,
		Slug:               row.Slug,
		Title:              row.Title,
		Category:           row.Category,
		SavesCount:         row.SavesCount,
		StopsCount:         row.StopsCount,
		Visibility:         row.Visibility,
		CoverPlaceID:       stringPtr(row.CoverPlaceID),
		CoverImageURL:      libraryCoverURL(stringPtr(row.CoverImageUrl)),
		CreatorUserID:      row.CreatorUserID,
		CreatorDisplayName: displayName,
		CreatorUsername:    row.CreatorUsername,
		CreatorAvatarURL:   row.CreatorAvatarUrl,
		IsMine:             row.CreatorUserID == viewerID,
		CreatedAt:          timestamptzToString(row.CreatedAt),
	}
}
