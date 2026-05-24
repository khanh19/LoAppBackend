package onboarding

import (
	"context"

	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

//encore:api auth method=POST path=/onboarding/complete
func (s *Service) CompleteOnboarding(ctx context.Context, req *CompleteOnboardingRequest) (*CompleteOnboardingResponse, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}
	if req == nil {
		req = &CompleteOnboardingRequest{}
	}

	if err := validateCompleteOnboardingRequest(req); err != nil {
		return nil, err
	}

	status, err := completeOnboarding(ctx, s.db, string(userID), req)
	if err != nil {
		return nil, err
	}

	return &CompleteOnboardingResponse{OnboardingStatus: status}, nil
}

func validateCompleteOnboardingRequest(req *CompleteOnboardingRequest) error {
	if req.Skip {
		return nil
	}

	if len(req.CityIDs) == 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "cityIds is required"}
	}
	if len(req.CategoryIDs) == 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "categoryIds is required"}
	}
	if len(req.CategoryIDs) > 3 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "select up to 3 categories"}
	}
	if len(req.VibeIDs) == 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "vibeIds is required"}
	}
	if len(req.PlaceIDs) == 0 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "placeIds is required"}
	}

	return nil
}
