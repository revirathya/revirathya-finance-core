package auth_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestTokenManagerIssueAndParse(t *testing.T) {
	tokens := auth.NewTokenManager(testSecret, time.Hour)
	userID := uuid.New()

	token, expiresAt, err := tokens.Issue(userID)
	if err != nil {
		t.Fatalf("issue: unexpected error: %v", err)
	}
	if until := time.Until(expiresAt); until < 59*time.Minute || until > time.Hour {
		t.Fatalf("expiresAt is %v from now, want ~1h", until)
	}

	got, err := tokens.Parse(token)
	if err != nil {
		t.Fatalf("parse: unexpected error: %v", err)
	}
	if got != userID {
		t.Fatalf("got user ID %v, want %v", got, userID)
	}
}

func TestTokenManagerParseRejects(t *testing.T) {
	tokens := auth.NewTokenManager(testSecret, time.Hour)
	userID := uuid.New()

	valid, _, err := tokens.Issue(userID)
	if err != nil {
		t.Fatalf("issue: unexpected error: %v", err)
	}

	expired, _, err := auth.NewTokenManager(testSecret, -time.Minute).Issue(userID)
	if err != nil {
		t.Fatalf("issue expired: unexpected error: %v", err)
	}

	otherSecret, _, err := auth.NewTokenManager("another-secret-another-secret-xx", time.Hour).Issue(userID)
	if err != nil {
		t.Fatalf("issue other secret: unexpected error: %v", err)
	}

	// Another user's payload with the original signature: the classic tampering attack.
	attacker, _, err := tokens.Issue(uuid.New())
	if err != nil {
		t.Fatalf("issue attacker: unexpected error: %v", err)
	}
	validParts, attackerParts := strings.Split(valid, "."), strings.Split(attacker, ".")
	tampered := validParts[0] + "." + attackerParts[1] + "." + validParts[2]

	claims := func(issuer, subject string) jwt.RegisteredClaims {
		return jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}
	}
	sign := func(method jwt.SigningMethod, key any, c jwt.RegisteredClaims) string {
		s, err := jwt.NewWithClaims(method, c).SignedString(key)
		if err != nil {
			t.Fatalf("sign: unexpected error: %v", err)
		}
		return s
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-jwt"},
		{"expired", expired},
		{"signed with another secret", otherSecret},
		{"payload swapped from another token", tampered},
		{"alg none", sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, claims("revirathya-finance-core", userID.String()))},
		{"HS512 instead of HS256", sign(jwt.SigningMethodHS512, []byte(testSecret), claims("revirathya-finance-core", userID.String()))},
		{"wrong issuer", sign(jwt.SigningMethodHS256, []byte(testSecret), claims("someone-else", userID.String()))},
		{"subject is not a UUID", sign(jwt.SigningMethodHS256, []byte(testSecret), claims("revirathya-finance-core", "not-a-uuid"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokens.Parse(tt.token)
			if !errors.Is(err, domain.ErrUnauthenticated) {
				t.Fatalf("got error %v, want domain.ErrUnauthenticated", err)
			}
			if got != uuid.Nil {
				t.Fatalf("got user ID %v, want uuid.Nil", got)
			}
		})
	}
}
