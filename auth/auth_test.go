package auth

import "testing"

func TestClaimBool(t *testing.T) {
	tests := []struct {
		name    string
		profile map[string]interface{}
		key     string
		want    bool
	}{
		{
			name: "true value",
			profile: map[string]interface{}{
				auth0IsNewAccountClaim: true,
			},
			key:  auth0IsNewAccountClaim,
			want: true,
		},
		{
			name: "false value",
			profile: map[string]interface{}{
				auth0IsNewAccountClaim: false,
			},
			key:  auth0IsNewAccountClaim,
			want: false,
		},
		{
			name:    "missing value",
			profile: map[string]interface{}{},
			key:     auth0IsNewAccountClaim,
			want:    false,
		},
		{
			name: "wrong type",
			profile: map[string]interface{}{
				auth0IsNewAccountClaim: "true",
			},
			key:  auth0IsNewAccountClaim,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claimBool(tt.profile, tt.key); got != tt.want {
				t.Fatalf("claimBool() = %v, want %v", got, tt.want)
			}
		})
	}
}
