package graphql_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revirathya/revirathya-finance-core/internal/auth"
	gqlDlv "github.com/revirathya/revirathya-finance-core/internal/delivery/graphql"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
)

func TestWithAuth(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	tokens := auth.NewTokenManager(secret, time.Hour)
	userID := uuid.New()

	valid, _, err := tokens.Issue(userID)
	if err != nil {
		t.Fatalf("issue: unexpected error: %v", err)
	}
	expired, _, err := auth.NewTokenManager(secret, -time.Minute).Issue(userID)
	if err != nil {
		t.Fatalf("issue expired: unexpected error: %v", err)
	}

	tests := []struct {
		name          string
		authorization string
		wantUserID    uuid.UUID // uuid.Nil means "no principal"
	}{
		{"valid token", "Bearer " + valid, userID},
		{"no header", "", uuid.Nil},
		{"invalid token", "Bearer garbage", uuid.Nil},
		{"expired token", "Bearer " + expired, uuid.Nil},
		{"wrong scheme", "Basic " + valid, uuid.Nil},
		{"missing scheme", valid, uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				called bool
				gotID  uuid.UUID
				gotErr error
			)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				gotID, gotErr = auth.UserIDFrom(r.Context())
			})

			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()

			gqlDlv.WithAuth(tokens, next).ServeHTTP(rec, req)

			if !called {
				t.Fatal("next handler was not called; requests must always pass through")
			}
			if tt.wantUserID == uuid.Nil {
				if !errors.Is(gotErr, domain.ErrUnauthenticated) {
					t.Fatalf("got principal %v (err %v), want no principal", gotID, gotErr)
				}
				return
			}
			if gotErr != nil {
				t.Fatalf("unexpected error: %v", gotErr)
			}
			if gotID != tt.wantUserID {
				t.Fatalf("got user ID %v, want %v", gotID, tt.wantUserID)
			}
		})
	}
}
