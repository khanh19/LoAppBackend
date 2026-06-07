package events

import (
	"context"
)

//encore:api public method=GET path=/events/upcoming
func (s *Service) ListUpcomingEvents(ctx context.Context, params *ListUpcomingEventsParams) (*ListUpcomingEventsResponse, error) {
	limit := defaultUpcomingLimit
	citySlug := ""
	if params != nil {
		limit = normalizeUpcomingLimit(params.Limit)
		citySlug = params.CitySlug
	}

	events, err := listUpcomingEvents(ctx, s.db, citySlug, limit)
	if err != nil {
		return nil, err
	}
	return &ListUpcomingEventsResponse{Events: events}, nil
}

//encore:api public method=GET path=/events
func (s *Service) ListEvents(ctx context.Context, params *ListEventsParams) (*ListEventsResponse, error) {
	limit := defaultListLimit
	offset := 0
	citySlug := ""
	if params != nil {
		limit = normalizeListLimit(params.Limit)
		offset = normalizeOffset(params.Offset)
		citySlug = params.CitySlug
	}

	events, _, hasMore, err := listExternalEvents(ctx, s.db, citySlug, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListEventsResponse{
		Events:  events,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}
