package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"encore.app/users"
	"encore.dev/beta/errs"
	"golang.org/x/oauth2"
)

func (s *Service) callbackResponseFromToken(ctx context.Context, token *oauth2.Token) (*CallbackResponse, error) {
	idToken, err := s.auth.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to verify ID Token.",
		}
	}

	var profile map[string]interface{}
	if err := idToken.Claims(&profile); err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: err.Error(),
		}
	}

	provider, providerUserID := parseAuth0Subject(profile["sub"])
	email := profileEmail(profile)
	emailVerified, _ := profile["email_verified"].(bool)
	rawProfile, err := json.Marshal(profile)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to encode profile data.",
		}
	}

	upserted, err := users.UpsertFromAuth(ctx, &users.UpsertFromAuthParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		EmailVerified:  emailVerified,
		RawProfile:     rawProfile,
	})
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Failed to persist authenticated user.",
		}
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: errors.New("missing id_token").Error(),
		}
	}

	return &CallbackResponse{
		Token:            rawIDToken,
		RefreshToken:     strings.TrimSpace(token.RefreshToken),
		ExpiresAt:        idToken.Expiry.Unix(),
		OnboardingStatus: upserted.OnboardingState,
		ProfileExists:    upserted.ProfileExists,
	}, nil
}
