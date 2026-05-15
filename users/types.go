package users

import (
	"encoding/json"
	"time"
)

type User struct {
	ID              string     `json:"id"`
	PrimaryEmail    *string    `json:"primary_email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	OnboardingState string     `json:"onboarding_status"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	IsNewIdentity   bool       `json:"is_new_identity,omitempty"`
}

type UpsertFromAuthParams struct {
	Provider       string          `json:"provider"`
	ProviderUserID string          `json:"provider_user_id"`
	Email          *string         `json:"email"`
	EmailVerified  bool            `json:"email_verified"`
	RawProfile     json.RawMessage `json:"raw_profile"`
}

type UpsertUserProfileRequest struct {
	FirstName           string  `json:"first_name"`
	LastName            string  `json:"last_name"`
	Username            string  `json:"username"`
	PhoneCountryCode    *string `json:"phone_country_code"`
	PhoneNationalNumber *string `json:"phone_national_number"`
	PhoneE164           *string `json:"-"`
	DateOfBirth         *string `json:"date_of_birth"`
	AvatarObjectKey     *string `json:"avatar_object_key"`
	AvatarURL           *string `json:"avatar_url"`
	Bio                 *string `json:"bio"`
}

type UserProfile struct {
	UserID              string    `json:"user_id"`
	FirstName           string    `json:"first_name"`
	LastName            string    `json:"last_name"`
	Username            string    `json:"username"`
	PhoneE164           *string   `json:"phone_e164"`
	PhoneCountryCode    *string   `json:"phone_country_code"`
	PhoneNationalNumber *string   `json:"phone_national_number"`
	DateOfBirth         *string   `json:"date_of_birth"`
	AvatarObjectKey     *string   `json:"avatar_object_key"`
	AvatarURL           *string   `json:"avatar_url"`
	Bio                 *string   `json:"bio"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type UsernameAvailabilityParams struct {
	Username string `query:"username"`
}

type UsernameAvailabilityResponse struct {
	Username  string `json:"username"`
	Available bool   `json:"available"`
}
