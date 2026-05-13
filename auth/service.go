package auth

// Service struct definition.
// Learn more: encore.dev/docs/primitives/services-and-apis/service-structs
//
//encore:service
type Service struct {
	auth *Authenticator
}

// initService is automatically called by Encore when the service starts up.
func initService() (*Service, error) {
	authenticator, err := New()
	if err != nil {
		return nil, err
	}
	return &Service{auth: authenticator}, nil
}
