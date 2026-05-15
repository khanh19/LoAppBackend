package users

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	encoreauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9_\.]+$`)
	e164Pattern     = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
	countryPattern  = regexp.MustCompile(`^\+[0-9]{1,4}$`)
)

//encore:api auth method=POST path=/users/profile
func (s *Service) UpsertProfile(ctx context.Context, req *UpsertUserProfileRequest) (*UserProfile, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}

	normalized, err := normalizeUpsertUserProfileRequest(req)
	if err != nil {
		return nil, err
	}

	profile, err := upsertUserProfile(ctx, s.db, string(userID), normalized)
	if err != nil {
		if isUsernameUniqueViolation(err) {
			return nil, &errs.Error{Code: errs.AlreadyExists, Message: "username is already taken"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to upsert user profile")
	}
	return profile, nil
}

//encore:api auth method=GET path=/users/profile
func (s *Service) GetProfile(ctx context.Context) (*UserProfile, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}

	profile, err := getUserProfile(ctx, s.db, string(userID))
	if err != nil {
		if errors.Is(err, errUserProfileNotFound) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "user profile not found"}
		}
		return nil, errs.WrapCode(err, errs.Internal, "failed to fetch user profile")
	}
	return profile, nil
}

//encore:api auth method=GET path=/users/username-availability
func (s *Service) CheckUsernameAvailability(ctx context.Context, params *UsernameAvailabilityParams) (*UsernameAvailabilityResponse, error) {
	userID, ok := encoreauth.UserID()
	if !ok {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "authentication required"}
	}
	if params == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "username is required"}
	}

	username, err := normalizeUsername(params.Username)
	if err != nil {
		return nil, err
	}

	available, err := isUsernameAvailable(ctx, s.db, username, string(userID))
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to check username availability")
	}
	return &UsernameAvailabilityResponse{Username: username, Available: available}, nil
}

func normalizeUpsertUserProfileRequest(req *UpsertUserProfileRequest) (*UpsertUserProfileRequest, error) {
	if req == nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "request is required"}
	}

	normalized := *req
	normalized.FirstName = strings.TrimSpace(normalized.FirstName)
	normalized.LastName = strings.TrimSpace(normalized.LastName)
	if normalized.FirstName == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "first_name is required"}
	}
	if normalized.LastName == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "last_name is required"}
	}
	if len(normalized.FirstName) > 80 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "first_name is too long"}
	}
	if len(normalized.LastName) > 80 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "last_name is too long"}
	}

	username, err := normalizeUsername(normalized.Username)
	if err != nil {
		return nil, err
	}
	normalized.Username = username

	normalized.AvatarObjectKey = normalizeOptionalString(normalized.AvatarObjectKey)
	normalized.AvatarURL = normalizeOptionalString(normalized.AvatarURL)
	normalized.Bio = normalizeOptionalString(normalized.Bio)

	phoneE164, country, national, err := normalizePhone(normalized.PhoneCountryCode, normalized.PhoneNationalNumber)
	if err != nil {
		return nil, err
	}
	normalized.PhoneE164 = phoneE164
	normalized.PhoneCountryCode = country
	normalized.PhoneNationalNumber = national
	normalized.DateOfBirth, err = normalizeDateOfBirth(normalized.DateOfBirth)
	if err != nil {
		return nil, err
	}

	return &normalized, nil
}

func normalizeUsername(raw string) (string, error) {
	username := strings.ToLower(strings.TrimSpace(raw))
	if username == "" {
		return "", &errs.Error{Code: errs.InvalidArgument, Message: "username is required"}
	}
	if len(username) < 3 || len(username) > 20 {
		return "", &errs.Error{Code: errs.InvalidArgument, Message: "username must be 3-20 characters"}
	}
	if !usernamePattern.MatchString(username) {
		return "", &errs.Error{Code: errs.InvalidArgument, Message: "username format is invalid"}
	}
	return username, nil
}

func normalizePhone(countryCode, nationalNumber *string) (phoneE164, normalizedCountry, normalizedNational *string, err error) {
	cc := trimPtr(countryCode)
	nn := trimPtr(nationalNumber)

	if cc == nil && nn == nil {
		return nil, nil, nil, nil
	}
	if cc == nil || nn == nil {
		return nil, nil, nil, &errs.Error{Code: errs.InvalidArgument, Message: "phone_country_code and phone_national_number must both be provided"}
	}
	if !countryPattern.MatchString(*cc) {
		return nil, nil, nil, &errs.Error{Code: errs.InvalidArgument, Message: "phone_country_code format is invalid"}
	}

	digits := digitsOnly(*nn)
	if digits == "" {
		return nil, nil, nil, &errs.Error{Code: errs.InvalidArgument, Message: "phone_national_number format is invalid"}
	}
	e164 := *cc + digits
	if !e164Pattern.MatchString(e164) {
		return nil, nil, nil, &errs.Error{Code: errs.InvalidArgument, Message: "phone number format is invalid"}
	}

	return ptr(e164), cc, ptr(digits), nil
}

func normalizeDateOfBirth(raw *string) (*string, error) {
	value := trimPtr(raw)
	if value == nil {
		return nil, nil
	}

	dob, err := time.Parse("2006-01-02", *value)
	if err != nil {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "date_of_birth must be YYYY-MM-DD"}
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if dob.After(today) {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "date_of_birth cannot be in the future"}
	}
	formatted := dob.Format("2006-01-02")
	return &formatted, nil
}

func normalizeOptionalString(value *string) *string {
	return trimPtr(value)
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func digitsOnly(v string) string {
	var b strings.Builder
	for _, r := range v {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ptr(v string) *string {
	return &v
}

func isUsernameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "user_profiles_username_key"
}
