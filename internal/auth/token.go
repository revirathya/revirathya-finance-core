package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

const tokenIssuer = "revirathya-finance-core" // nolint:gosec // G101: issuer name, not a credential

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Issue returns a signed session token for userID and its expiry time.
func (m *TokenManager) Issue(userId uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)
	claims := jwt.RegisteredClaims{
		Subject:   userId.String(),
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign session token: %w", err)
	}

	return token, expiresAt, nil
}

// Parse verifies a session token and returns its user ID.
// Any invalid, expired or tampered token returns errDom.ErrUnauthenticated.
func (m *TokenManager) Parse(token string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(5*time.Second),
	)

	if err != nil {
		return uuid.Nil, errDom.ErrUnauthenticated
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, errDom.ErrUnauthenticated
	}

	return userId, nil
}
