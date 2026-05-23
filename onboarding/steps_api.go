package onboarding

import (
	"context"

	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

//encore:api auth method=POST path=/onboarding/steps/:step
func (s *Service) SaveStep(ctx context.Context, step int, req *SaveStepRequest) (*SaveStepResponse, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}

	if err := validateSaveStepRequest(step, req); err != nil {
		return nil, err
	}

	if err := saveOnboardingStep(ctx, s.db, string(userID), step, req); err != nil {
		return nil, err
	}

	return &SaveStepResponse{Step: step, Completed: true}, nil
}

//encore:api auth method=POST path=/onboarding/complete
func (s *Service) CompleteOnboarding(ctx context.Context, req *CompleteOnboardingRequest) (*CompleteOnboardingResponse, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}

	skip := req != nil && req.Skip
	status, err := completeOnboarding(ctx, s.db, string(userID), skip)
	if err != nil {
		return nil, err
	}

	return &CompleteOnboardingResponse{OnboardingStatus: status}, nil
}

func validateSaveStepRequest(step int, req *SaveStepRequest) error {
	if step < 1 || step > 4 {
		return &errs.Error{Code: errs.InvalidArgument, Message: "step must be between 1 and 4"}
	}

	switch step {
	case 1:
		if len(req.CityIDs) == 0 {
			return &errs.Error{Code: errs.InvalidArgument, Message: "cityIds is required for step 1"}
		}
	case 2:
		if len(req.CategoryIDs) == 0 {
			return &errs.Error{Code: errs.InvalidArgument, Message: "categoryIds is required for step 2"}
		}
		if len(req.CategoryIDs) > 3 {
			return &errs.Error{Code: errs.InvalidArgument, Message: "select up to 3 categories"}
		}
	case 3:
		if len(req.VibeIDs) == 0 {
			return &errs.Error{Code: errs.InvalidArgument, Message: "vibeIds is required for step 3"}
		}
	case 4:
		if len(req.PlaceIDs) == 0 {
			return &errs.Error{Code: errs.InvalidArgument, Message: "placeIds is required for step 4"}
		}
	}

	return nil
}
