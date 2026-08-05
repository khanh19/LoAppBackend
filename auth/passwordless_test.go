package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"encore.dev/beta/errs"
	"golang.org/x/oauth2"
)

func TestStartEmailOTPSendsAuth0PasswordlessRequest(t *testing.T) {
	var got passwordlessStartRequest
	var gotForwardedFor string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/passwordless/start" {
			t.Fatalf("path = %q, want /passwordless/start", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		gotForwardedFor = r.Header.Get("auth0-forwarded-for")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	authenticator := testPasswordlessAuthenticator(server, "client-id", "client-secret")
	if err := authenticator.StartEmailOTP(context.Background(), "user@example.com", "203.0.113.7"); err != nil {
		t.Fatal(err)
	}

	if got.ClientID != "client-id" {
		t.Errorf("client_id = %q", got.ClientID)
	}
	if got.ClientSecret != "client-secret" {
		t.Errorf("client_secret = %q", got.ClientSecret)
	}
	if got.Connection != "email" {
		t.Errorf("connection = %q", got.Connection)
	}
	if got.Email != "user@example.com" {
		t.Errorf("email = %q", got.Email)
	}
	if got.Send != "code" {
		t.Errorf("send = %q", got.Send)
	}
	if got.AuthParams.Scope != "openid profile email offline_access" {
		t.Errorf("scope = %q", got.AuthParams.Scope)
	}
	if gotForwardedFor != "203.0.113.7" {
		t.Errorf("auth0-forwarded-for = %q", gotForwardedFor)
	}
}

func TestExchangeEmailOTPReturnsOAuthTokenWithIDTokenExtra(t *testing.T) {
	var got passwordlessTokenRequest
	var gotForwardedFor string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("path = %q, want /oauth/token", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		gotForwardedFor = r.Header.Get("auth0-forwarded-for")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token":"access-token",
			"refresh_token":"refresh-token",
			"id_token":"id-token",
			"token_type":"Bearer",
			"expires_in":3600
		}`))
	}))
	defer server.Close()

	authenticator := testPasswordlessAuthenticator(server, "client-id", "client-secret")
	token, err := authenticator.ExchangeEmailOTP(context.Background(), "user@example.com", "123456", "203.0.113.9")
	if err != nil {
		t.Fatal(err)
	}

	if got.GrantType != "http://auth0.com/oauth/grant-type/passwordless/otp" {
		t.Errorf("grant_type = %q", got.GrantType)
	}
	if got.ClientID != "client-id" || got.ClientSecret != "client-secret" {
		t.Errorf("client credentials = %q/%q", got.ClientID, got.ClientSecret)
	}
	if got.Username != "user@example.com" {
		t.Errorf("username = %q", got.Username)
	}
	if got.OTP != "123456" {
		t.Errorf("otp = %q", got.OTP)
	}
	if got.Realm != "email" {
		t.Errorf("realm = %q", got.Realm)
	}
	if token.AccessToken != "access-token" || token.RefreshToken != "refresh-token" || token.TokenType != "Bearer" {
		t.Errorf("token = %#v", token)
	}
	if idToken, _ := token.Extra("id_token").(string); idToken != "id-token" {
		t.Errorf("id_token extra = %q", idToken)
	}
	if token.Expiry.IsZero() {
		t.Error("expiry was not set")
	}
	if gotForwardedFor != "203.0.113.9" {
		t.Errorf("auth0-forwarded-for = %q", gotForwardedFor)
	}
}

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail(" USER@Example.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "user@example.com" {
		t.Errorf("normalizeEmail = %q", got)
	}

	if _, err := normalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestNormalizeForwardedFor(t *testing.T) {
	if got := normalizeForwardedFor("203.0.113.10, 10.0.0.1"); got != "203.0.113.10" {
		t.Errorf("normalizeForwardedFor = %q", got)
	}
	if got := normalizeForwardedFor("not-an-ip"); got != "" {
		t.Errorf("normalizeForwardedFor invalid = %q", got)
	}
}

func TestAuth0StartOTPErrorMapsRateLimit(t *testing.T) {
	err := auth0StartOTPError(&auth0APIError{
		StatusCode: http.StatusTooManyRequests,
		Code:       "too_many_attempts",
		Message:    "too many attempts",
		RetryAfter: "60",
	})

	var encoreErr *errs.Error
	if !errors.As(err, &encoreErr) {
		t.Fatalf("error = %T, want *errs.Error", err)
	}
	if encoreErr.Code != errs.ResourceExhausted {
		t.Errorf("code = %v, want resource exhausted", encoreErr.Code)
	}
	if encoreErr.Message != "Too many verification code requests. Try again later." {
		t.Errorf("message = %q", encoreErr.Message)
	}
	details, ok := encoreErr.Details.(auth0ErrorDetails)
	if !ok {
		t.Fatalf("details = %T, want auth0ErrorDetails", encoreErr.Details)
	}
	if details.Error != "too_many_attempts" || details.RetryAfter != "60" || details.Reason != "too many attempts" {
		t.Errorf("details = %#v", details)
	}
}

func TestAuth0VerifyOTPErrorMapsUnauthorizedClient(t *testing.T) {
	err := auth0VerifyOTPError(&auth0APIError{
		StatusCode: http.StatusForbidden,
		Code:       "unauthorized_client",
		Message:    "Grant type not allowed for the client.",
	})

	var encoreErr *errs.Error
	if !errors.As(err, &encoreErr) {
		t.Fatalf("error = %T, want *errs.Error", err)
	}
	if encoreErr.Code != errs.FailedPrecondition {
		t.Errorf("code = %v, want failed precondition", encoreErr.Code)
	}
	details, ok := encoreErr.Details.(auth0ErrorDetails)
	if !ok {
		t.Fatalf("details = %T, want auth0ErrorDetails", encoreErr.Details)
	}
	if details.Error != "unauthorized_client" || details.Reason != "Grant type not allowed for the client." {
		t.Errorf("details = %#v", details)
	}
}

func TestParseAuth0SubjectPasswordlessEmail(t *testing.T) {
	provider, providerUserID := parseAuth0Subject("email|user@example.com")
	if provider != "email" {
		t.Errorf("provider = %q", provider)
	}
	if providerUserID != "user@example.com" {
		t.Errorf("providerUserID = %q", providerUserID)
	}
}

func testPasswordlessAuthenticator(server *httptest.Server, clientID, clientSecret string) *Authenticator {
	return &Authenticator{
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		auth0Domain: server.URL,
		httpClient:  server.Client(),
	}
}
