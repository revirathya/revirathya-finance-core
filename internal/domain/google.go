package domain

import "context"

// Entity
type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	GivenName     string
	FamilyName    string
}

// Repository
type GoogleVerifier interface {
	VerifyIdentity(ctx context.Context, idToken string) (*GoogleIdentity, error)
}
