package users

import (
	"context"
	"encoding/json"
	"strings"

	"encore.dev/beta/errs"
)

//encore:api private method=POST path=/users/upsert-from-auth
func (s *Service) UpsertFromAuth(ctx context.Context, req *UpsertFromAuthParams) (*User, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}

	normalized, rawProfile, err := normalizeUpsertFromAuthRequest(req)
	if err != nil {
		return nil, err
	}

	return s.upsertFromAuth(ctx, normalized, rawProfile)
}

func normalizeUpsertFromAuthRequest(req *UpsertFromAuthParams) (*UpsertFromAuthParams, json.RawMessage, error) {
	normalized := *req
	normalized.Provider = strings.TrimSpace(normalized.Provider)
	normalized.ProviderUserID = strings.TrimSpace(normalized.ProviderUserID)
	if normalized.Provider == "" || normalized.ProviderUserID == "" {
		return nil, nil, &errs.Error{Code: errs.InvalidArgument, Message: "provider and provider_user_id are required"}
	}

	if normalized.Email != nil {
		email := strings.TrimSpace(*normalized.Email)
		if email == "" {
			normalized.Email = nil
		} else {
			normalized.Email = &email
		}
	}

	rawProfile := normalized.RawProfile
	if len(rawProfile) == 0 {
		rawProfile = json.RawMessage(`{}`)
	}
	if !json.Valid(rawProfile) {
		return nil, nil, &errs.Error{Code: errs.InvalidArgument, Message: "raw_profile must be valid JSON"}
	}

	return &normalized, rawProfile, nil
}
