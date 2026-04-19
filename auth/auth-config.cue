ClientID: "xkRK6mYn9bVq1S8ASiwKmT2oWwX2vtAs"
Domain:   "dev-nnz0pmv3onjn82oq.us.auth0.com"

// CUE unifies all matching blocks. Do not set the same field twice: use one branch
// for local `encore run` and another for every other env.

// `encore run` — web/curl/browser tests (`expo start --web` or paste auth_code_url in browser).
if #Meta.Environment.Type == "development" && #Meta.Environment.Cloud == "local" {
	CallbackURL: "http://localhost:8081/callback"
	LogoutURL:   "http://localhost:8081/"
}

// Encore Cloud, EAS builds, native dev client — Auth0 redirects to the app scheme.
if #Meta.Environment.Type != "development" || #Meta.Environment.Cloud != "local" {
	CallbackURL: "loapp://callback"
	LogoutURL:   "loapp://"
}
