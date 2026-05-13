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
