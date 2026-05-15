package auth

type LoginResponse struct {
	State       string `json:"state"`
	AuthCodeURL string `json:"auth_code_url"`
}

type CallbackRequest struct {
	Code string `json:"code"`
}

type CallbackResponse struct {
	Token            string `json:"token"`
	OnboardingStatus string `json:"onboarding_status"`
	ProfileExists    bool   `json:"profile_exists"`
}

type OTPStartRequest struct {
	Email        string `json:"email"`
	ForwardedFor string `header:"X-Forwarded-For" json:"-"`
}

type OTPStartResponse struct {
	Sent bool `json:"sent"`
}

type OTPVerifyRequest struct {
	Email        string `json:"email"`
	Code         string `json:"code"`
	ForwardedFor string `header:"X-Forwarded-For" json:"-"`
}

type LogoutResponse struct {
	RedirectURL string `json:"redirect_url"`
}

type ProfileData struct {
	Email   string `json:"email"`
	Picture string `json:"picture"`
}
