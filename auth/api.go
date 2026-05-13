package auth

import (
	"context"
	"net/url"
	"strings"

	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

//encore:api public method=POST path=/auth/login
func (s *Service) Login(ctx context.Context) (*LoginResponse, error) {
	state, err := generateRandomState()
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: err.Error(),
		}
	}

	return &LoginResponse{
		State:       state,
		AuthCodeURL: s.auth.AuthCodeURL(state),
	}, nil
}

//encore:api public method=POST path=/auth/callback
func (s *Service) Callback(ctx context.Context, req *CallbackRequest) (*CallbackResponse, error) {
	token, err := s.auth.Exchange(ctx, req.Code)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.PermissionDenied,
			Message: "Failed to convert an authorization code into a token.",
		}
	}

	return s.callbackResponseFromToken(ctx, token)
}

//encore:api public method=GET path=/auth/logout
func (s *Service) Logout(ctx context.Context) (*LogoutResponse, error) {
	logoutURL, err := url.Parse("https://" + cfg.Domain() + "/v2/logout")
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: err.Error(),
		}
	}

	returnTo, err := url.Parse(cfg.LogoutURL())
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: err.Error(),
		}
	}

	parameters := url.Values{}
	parameters.Add("returnTo", returnTo.String())
	parameters.Add("client_id", cfg.ClientID())
	logoutURL.RawQuery = parameters.Encode()

	return &LogoutResponse{
		RedirectURL: logoutURL.String(),
	}, nil
}

// Endpoints annotated with `auth` are public and requires authentication
// Learn more: encore.dev/docs/primitives/apis#access-controls
//
//encore:api auth method=GET path=/profile
func GetProfile(ctx context.Context) (*ProfileData, error) {
	return encoreauth.Data().(*ProfileData), nil
}

func profileEmail(profile map[string]interface{}) *string {
	v, ok := profile["email"].(string)
	if !ok {
		return nil
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
