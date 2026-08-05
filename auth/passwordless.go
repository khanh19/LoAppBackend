package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type passwordlessStartRequest struct {
	ClientID     string                    `json:"client_id"`
	ClientSecret string                    `json:"client_secret,omitempty"`
	Connection   string                    `json:"connection"`
	Email        string                    `json:"email"`
	Send         string                    `json:"send"`
	AuthParams   passwordlessStartAuthData `json:"authParams"`
}

type passwordlessStartAuthData struct {
	Scope string `json:"scope"`
}

type passwordlessTokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	Username     string `json:"username"`
	OTP          string `json:"otp"`
	Realm        string `json:"realm"`
	Scope        string `json:"scope"`
}

type passwordlessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type auth0ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"message"`
}

type auth0APIError struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter string
}

func (e *auth0APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("auth0 request failed (%d)", e.StatusCode)
	}
	return fmt.Sprintf("auth0 request failed (%d): %s", e.StatusCode, e.Message)
}

func (a *Authenticator) StartEmailOTP(ctx context.Context, email, forwardedFor string) error {
	req := passwordlessStartRequest{
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
		Connection:   "email",
		Email:        email,
		Send:         "code",
		AuthParams: passwordlessStartAuthData{
			Scope: "openid profile email offline_access",
		},
	}

	return a.postAuth0JSON(ctx, "/passwordless/start", req, forwardedFor, nil)
}

func (a *Authenticator) ExchangeEmailOTP(ctx context.Context, email, code, forwardedFor string) (*oauth2.Token, error) {
	req := passwordlessTokenRequest{
		GrantType:    "http://auth0.com/oauth/grant-type/passwordless/otp",
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
		Username:     email,
		OTP:          code,
		Realm:        "email",
		Scope:        "openid profile email offline_access",
	}

	var resp passwordlessTokenResponse
	if err := a.postAuth0JSON(ctx, "/oauth/token", req, forwardedFor, &resp); err != nil {
		return nil, err
	}
	if resp.IDToken == "" {
		return nil, errors.New("auth0 passwordless response missing id_token")
	}

	token := &oauth2.Token{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
	}
	if resp.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	}

	return token.WithExtra(map[string]interface{}{"id_token": resp.IDToken}), nil
}

func (a *Authenticator) postAuth0JSON(ctx context.Context, path string, payload interface{}, forwardedFor string, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.auth0URL(path), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if forwardedFor != "" {
		req.Header.Set("auth0-forwarded-for", forwardedFor)
	}

	client := a.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return newAuth0APIError(resp.StatusCode, resp.Header.Get("Retry-After"), respBody)
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	return json.Unmarshal(respBody, out)
}

func (a *Authenticator) auth0URL(path string) string {
	domain := strings.TrimRight(a.auth0Domain, "/")
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return domain + path
	}
	return "https://" + domain + path
}

func newAuth0APIError(statusCode int, retryAfter string, body []byte) error {
	var auth0Err auth0ErrorResponse
	if err := json.Unmarshal(body, &auth0Err); err == nil {
		switch {
		case auth0Err.ErrorDescription != "":
			return &auth0APIError{StatusCode: statusCode, Code: auth0Err.Error, Message: auth0Err.ErrorDescription, RetryAfter: retryAfter}
		case auth0Err.Message != "":
			return &auth0APIError{StatusCode: statusCode, Code: auth0Err.Error, Message: auth0Err.Message, RetryAfter: retryAfter}
		case auth0Err.Error != "":
			return &auth0APIError{StatusCode: statusCode, Code: auth0Err.Error, Message: auth0Err.Error, RetryAfter: retryAfter}
		}
	}
	return &auth0APIError{StatusCode: statusCode, RetryAfter: retryAfter}
}
