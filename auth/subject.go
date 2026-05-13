package auth

import "strings"

func parseAuth0Subject(raw interface{}) (provider string, providerUserID string) {
	sub, ok := raw.(string)
	if !ok || strings.TrimSpace(sub) == "" {
		return "auth0", ""
	}

	parts := strings.SplitN(sub, "|", 2)
	prefix := strings.TrimSpace(parts[0])
	switch prefix {
	case "google-oauth2":
		provider = "google"
	case "apple":
		provider = "apple"
	case "facebook":
		provider = "facebook"
	case "email":
		provider = "email"
	case "auth0":
		provider = "email"
	default:
		provider = "auth0"
	}

	if len(parts) == 2 {
		return provider, strings.TrimSpace(parts[1])
	}
	return provider, strings.TrimSpace(sub)
}
