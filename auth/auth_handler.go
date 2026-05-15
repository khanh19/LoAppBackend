package auth

import (
	"context"
	"encoding/json"

	"encore.app/users"
	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/coreos/go-oidc/v3/oidc"
)

// The `encore:authhandler` annotation tells Encore to run this function for all
// incoming API call that requires authentication.
// Learn more: encore.dev/docs/develop/auth#the-auth-handler
//
//encore:authhandler
func (s *Service) AuthHandler(ctx context.Context, token string) (encoreauth.UID, *ProfileData, error) {
	oidcConfig := &oidc.Config{
		ClientID: s.auth.ClientID,
	}

	t, err := s.auth.Verifier(oidcConfig).Verify(ctx, token)
	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: "invalid token",
		}
	}

	var profile map[string]interface{}
	if err := t.Claims(&profile); err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Internal,
			Message: err.Error(),
		}
	}

	provider, providerUserID := parseAuth0Subject(profile["sub"])
	email := profileEmail(profile)
	emailVerified, _ := profile["email_verified"].(bool)
	rawProfile, err := json.Marshal(profile)
	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Internal,
			Message: "failed to encode profile data",
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
		return "", nil, &errs.Error{
			Code:    errs.Internal,
			Message: "failed to persist authenticated user",
		}
	}

	profileData := &ProfileData{
		Email:   valueOrEmpty(profile["email"]),
		Picture: valueOrEmpty(profile["picture"]),
	}

	return encoreauth.UID(upserted.ID), profileData, nil
}

func valueOrEmpty(v interface{}) string {
	s, _ := v.(string)
	return s
}
