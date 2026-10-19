package testutil

import (
	"context"
	"sync"

	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

// FakeGoogleVerifier implements domain.GoogleVerifier: Add an ID token with the identity Google would return for
// it; any other token is rejected as unauthenticated, like an invalid or expired Google token.
type FakeGoogleVerifier struct {
	mu         sync.Mutex
	identities map[string]domain.GoogleIdentity
}

func NewFakeGoogleVerifier() *FakeGoogleVerifier {
	return &FakeGoogleVerifier{identities: map[string]domain.GoogleIdentity{}}
}

func (f *FakeGoogleVerifier) Add(idToken string, identity domain.GoogleIdentity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.identities[idToken] = identity
}

func (f *FakeGoogleVerifier) VerifyIdentity(_ context.Context, idToken string) (*domain.GoogleIdentity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	identity, ok := f.identities[idToken]
	if !ok {
		return nil, errDom.ErrUnauthenticated
	}
	return &identity, nil
}
