package stamps

import (
	"context"

	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

func requireUserID() (string, error) {
	userID, ok := encoreauth.UserID()
	if !ok || string(userID) == "" {
		return "", &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}
	return string(userID), nil
}

//encore:api auth method=POST path=/stamps
func (s *Service) CreateStamp(ctx context.Context, req *CreateStampRequest) (*CreateStampResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	return s.createStamp(ctx, userID, req)
}

//encore:api auth method=POST path=/stamps/:id/pairwise
func (s *Service) SubmitStampPairwise(ctx context.Context, id string, req *SubmitPairwiseRequest) (*SubmitPairwiseResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	return s.submitPairwise(ctx, userID, id, req)
}

//encore:api auth method=GET path=/me/pool/:category
func (s *Service) GetMyPool(ctx context.Context, category string) (*GetPoolResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	return s.getPool(ctx, userID, category)
}

//encore:api auth method=POST path=/me/familiarity
func (s *Service) SetFamiliarity(ctx context.Context, req *FamiliarityRequest) (*FamiliarityResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	return s.setFamiliarity(ctx, userID, req)
}

//encore:api auth method=GET path=/me/pairwise/next
func (s *Service) GetNextPairwise(ctx context.Context) (*NextPairwiseResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	return s.nextPairwise(ctx, userID)
}

//encore:api auth method=POST path=/pairwise
func (s *Service) SubmitPairwise(ctx context.Context, req *StandalonePairwiseRequest) (*SubmitPairwiseResponse, error) {
	userID, err := requireUserID()
	if err != nil {
		return nil, err
	}
	return s.submitStandalonePairwise(ctx, userID, req)
}
