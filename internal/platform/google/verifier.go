package google

import (
	"context"

	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"google.golang.org/api/idtoken"
)

type GoogleVerifier struct {
	clientID string
}

func NewGoogleVerifier(clientID string) domain.GoogleVerifier {
	return &GoogleVerifier{
		clientID: clientID,
	}
}

func (v *GoogleVerifier) VerifyIdentity(ctx context.Context, idToken string) (*domain.GoogleIdentity, error) {
	payload, err := idtoken.Validate(ctx, idToken, v.clientID)
	if err != nil {
		return nil, domain.ErrUnauthenticated
	}

	if (payload.Issuer != "accounts.google.com") && (payload.Issuer != "https://accounts.google.com") {
		return nil, domain.ErrUnauthenticated
	}

	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	givenName, _ := payload.Claims["given_name"].(string)
	familyName, _ := payload.Claims["family_name"].(string)

	result := &domain.GoogleIdentity{
		Subject:       payload.Subject,
		Email:         email,
		EmailVerified: emailVerified,
		GivenName:     givenName,
		FamilyName:    familyName,
	}
	return result, nil
}
