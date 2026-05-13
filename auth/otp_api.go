package auth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"

	"encore.dev/beta/errs"
)

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

//encore:api public method=POST path=/auth/otp/start
func (s *Service) StartEmailOTP(ctx context.Context, req *OTPStartRequest) (*OTPStartResponse, error) {
	email, err := normalizeEmail(startRequestEmail(req))
	if err != nil {
		return nil, err
	}

	if err := s.auth.StartEmailOTP(ctx, email, normalizeForwardedFor(reqForwardedFor(req))); err != nil {
		return nil, auth0StartOTPError(err)
	}

	return &OTPStartResponse{Sent: true}, nil
}

type auth0ErrorDetails struct {
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
	RetryAfter string `json:"retry_after,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func (auth0ErrorDetails) ErrDetails() {}

func auth0StartOTPError(err error) error {
	var auth0Err *auth0APIError
	if !errors.As(err, &auth0Err) {
		return &errs.Error{
			Code:    errs.Unavailable,
			Message: "Unable to send verification code.",
			Meta:    errs.Metadata{"cause": err.Error()},
		}
	}

	details := auth0ErrorDetails{
		StatusCode: auth0Err.StatusCode,
		Error:      auth0Err.Code,
		RetryAfter: auth0Err.RetryAfter,
		Reason:     auth0Err.Message,
	}

	switch auth0Err.StatusCode {
	case http.StatusTooManyRequests:
		return &errs.Error{
			Code:    errs.ResourceExhausted,
			Message: "Too many verification code requests. Try again later.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	case http.StatusBadRequest:
		return &errs.Error{
			Code:    errs.InvalidArgument,
			Message: "Unable to send verification code for this email.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return &errs.Error{
			Code:    errs.FailedPrecondition,
			Message: "Auth0 passwordless email is not configured correctly.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	default:
		return &errs.Error{
			Code:    errs.Unavailable,
			Message: "Unable to send verification code.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	}
}

//encore:api public method=POST path=/auth/otp/verify
func (s *Service) VerifyEmailOTP(ctx context.Context, req *OTPVerifyRequest) (*CallbackResponse, error) {
	email, err := normalizeEmail(verifyRequestEmail(req))
	if err != nil {
		return nil, err
	}
	code := normalizeOTPCode(reqCode(req))
	if code == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "code is required"}
	}

	token, err := s.auth.ExchangeEmailOTP(ctx, email, code, normalizeForwardedFor(reqForwardedFor(req)))
	if err != nil {
		return nil, auth0VerifyOTPError(err)
	}

	return s.callbackResponseFromToken(ctx, token)
}

func auth0VerifyOTPError(err error) error {
	var auth0Err *auth0APIError
	if !errors.As(err, &auth0Err) {
		return &errs.Error{
			Code:    errs.Unavailable,
			Message: "Unable to verify code.",
			Meta:    errs.Metadata{"cause": err.Error()},
		}
	}

	details := auth0ErrorDetails{
		StatusCode: auth0Err.StatusCode,
		Error:      auth0Err.Code,
		RetryAfter: auth0Err.RetryAfter,
		Reason:     auth0Err.Message,
	}

	switch auth0Err.Code {
	case "unauthorized_client", "access_denied":
		return &errs.Error{
			Code:    errs.FailedPrecondition,
			Message: "Auth0 passwordless OTP grant is not configured correctly.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	case "invalid_grant":
		return &errs.Error{
			Code:    errs.PermissionDenied,
			Message: "Invalid or expired verification code.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	}

	switch auth0Err.StatusCode {
	case http.StatusTooManyRequests:
		return &errs.Error{
			Code:    errs.ResourceExhausted,
			Message: "Too many verification attempts. Request a new code later.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	case http.StatusBadRequest, http.StatusForbidden:
		return &errs.Error{
			Code:    errs.PermissionDenied,
			Message: "Invalid or expired verification code.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	case http.StatusUnauthorized:
		return &errs.Error{
			Code:    errs.FailedPrecondition,
			Message: "Auth0 passwordless OTP grant is not configured correctly.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	default:
		return &errs.Error{
			Code:    errs.Unavailable,
			Message: "Unable to verify code.",
			Details: details,
			Meta:    errs.Metadata{"auth0_error": auth0Err.Error()},
		}
	}
}

func startRequestEmail(req *OTPStartRequest) string {
	if req == nil {
		return ""
	}
	return req.Email
}

func verifyRequestEmail(req *OTPVerifyRequest) string {
	if req == nil {
		return ""
	}
	return req.Email
}

func reqForwardedFor(req interface{}) string {
	switch v := req.(type) {
	case *OTPStartRequest:
		if v == nil {
			return ""
		}
		return v.ForwardedFor
	case *OTPVerifyRequest:
		if v == nil {
			return ""
		}
		return v.ForwardedFor
	default:
		return ""
	}
}

func reqCode(req *OTPVerifyRequest) string {
	if req == nil {
		return ""
	}
	return req.Code
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", &errs.Error{Code: errs.InvalidArgument, Message: "email is required"}
	}
	if !emailPattern.MatchString(email) {
		return "", &errs.Error{Code: errs.InvalidArgument, Message: "email must be valid"}
	}
	return strings.ToLower(email), nil
}

func normalizeOTPCode(code string) string {
	return strings.TrimSpace(code)
}

func normalizeForwardedFor(forwardedFor string) string {
	ip := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
