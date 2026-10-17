package auth_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/auth"
)

// Fake: Google verifier
type fakeVerifier struct {
	identity *domain.GoogleIdentity
	err      error

	gotToken string
}

func (f *fakeVerifier) VerifyIdentity(ctx context.Context, idToken string) (*domain.GoogleIdentity, error) {
	f.gotToken = idToken
	return f.identity, f.err
}

// Fake: user repository. GetUserByGoogleSub returns lookups[i] on its i-th call,
// so tests can model "not found, then found" for the registration race.
type lookupResult struct {
	user *domain.User
	err  error
}

type fakeUserRepo struct {
	lookups    []lookupResult
	createResp *domain.User
	createErr  error

	lookupCalls   int
	createCalls   int
	gotCreateForm domain.UserForm
}

func (f *fakeUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, errors.New("not used")
}

func (f *fakeUserRepo) GetUserByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	r := f.lookups[f.lookupCalls]
	f.lookupCalls++
	return r.user, r.err
}

func (f *fakeUserRepo) CreateUser(ctx context.Context, form domain.UserForm) (*domain.User, error) {
	f.createCalls++
	f.gotCreateForm = form
	return f.createResp, f.createErr
}

func (f *fakeUserRepo) UpdateUser(ctx context.Context, id uuid.UUID, firstName string, lastName *string) (*domain.User, error) {
	return nil, errors.New("not used")
}

// Test
func TestLoginWithGoogle(t *testing.T) {
	identity := func() *domain.GoogleIdentity {
		return &domain.GoogleIdentity{
			Subject:       "google-sub-123",
			Email:         "avi@example.com",
			EmailVerified: true,
			GivenName:     "Avi",
			FamilyName:    "Leonardi",
		}
	}
	existing := &domain.User{ID: uuid.New(), FirstName: "Avi", Email: "avi@example.com"}
	created := &domain.User{ID: uuid.New(), FirstName: "Avi", Email: "avi@example.com"}

	t.Run("returns existing user without creating", func(t *testing.T) {
		verifier := &fakeVerifier{identity: identity()}
		repo := &fakeUserRepo{lookups: []lookupResult{{user: existing}}}
		uc := auth.NewAuthUsecase(verifier, repo)

		got, err := uc.LoginWithGoogle(context.Background(), "id-token")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != existing {
			t.Fatalf("got %v, want %v", got, existing)
		}
		if repo.createCalls != 0 {
			t.Fatalf("CreateUser called %d times, want 0", repo.createCalls)
		}
		if verifier.gotToken != "id-token" {
			t.Fatalf("verifier received %q, want %q", verifier.gotToken, "id-token")
		}
	})

	t.Run("creates new user from Google identity", func(t *testing.T) {
		verifier := &fakeVerifier{identity: identity()}
		repo := &fakeUserRepo{
			lookups:    []lookupResult{{err: domain.ErrNotFound}},
			createResp: created,
		}
		uc := auth.NewAuthUsecase(verifier, repo)

		got, err := uc.LoginWithGoogle(context.Background(), "id-token")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != created {
			t.Fatalf("got %v, want %v", got, created)
		}
		lastName := "Leonardi"
		wantForm := domain.UserForm{
			FirstName: "Avi",
			LastName:  &lastName,
			Email:     "avi@example.com",
			GoogleSub: "google-sub-123",
		}
		if !reflect.DeepEqual(repo.gotCreateForm, wantForm) {
			t.Fatalf("repo received form %+v, want %+v", repo.gotCreateForm, wantForm)
		}
	})

	t.Run("falls back to email local part and nil last name", func(t *testing.T) {
		id := identity()
		id.GivenName = ""
		id.FamilyName = ""
		verifier := &fakeVerifier{identity: id}
		repo := &fakeUserRepo{
			lookups:    []lookupResult{{err: domain.ErrNotFound}},
			createResp: created,
		}
		uc := auth.NewAuthUsecase(verifier, repo)

		if _, err := uc.LoginWithGoogle(context.Background(), "id-token"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotCreateForm.FirstName != "avi" {
			t.Fatalf("first name %q, want %q", repo.gotCreateForm.FirstName, "avi")
		}
		if repo.gotCreateForm.LastName != nil {
			t.Fatalf("last name %q, want nil", *repo.gotCreateForm.LastName)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		verifier := &fakeVerifier{err: domain.ErrUnauthenticated}
		repo := &fakeUserRepo{}
		uc := auth.NewAuthUsecase(verifier, repo)

		if _, err := uc.LoginWithGoogle(context.Background(), "bad"); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("got %v, want domain.ErrUnauthenticated", err)
		}
		if repo.lookupCalls != 0 || repo.createCalls != 0 {
			t.Fatal("repository must not be called for an invalid token")
		}
	})

	t.Run("rejects unverified email", func(t *testing.T) {
		id := identity()
		id.EmailVerified = false
		verifier := &fakeVerifier{identity: id}
		repo := &fakeUserRepo{}
		uc := auth.NewAuthUsecase(verifier, repo)

		if _, err := uc.LoginWithGoogle(context.Background(), "id-token"); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("got %v, want domain.ErrUnauthenticated", err)
		}
		if repo.lookupCalls != 0 || repo.createCalls != 0 {
			t.Fatal("repository must not be called for an unverified email")
		}
	})

	t.Run("lookup failure does not create a user", func(t *testing.T) {
		dbErr := errors.New("db down")
		verifier := &fakeVerifier{identity: identity()}
		repo := &fakeUserRepo{lookups: []lookupResult{{err: dbErr}}}
		uc := auth.NewAuthUsecase(verifier, repo)

		if _, err := uc.LoginWithGoogle(context.Background(), "id-token"); !errors.Is(err, dbErr) {
			t.Fatalf("got %v, want %v", err, dbErr)
		}
		if repo.createCalls != 0 {
			t.Fatalf("CreateUser called %d times, want 0", repo.createCalls)
		}
	})

	t.Run("registration race returns the user created concurrently", func(t *testing.T) {
		verifier := &fakeVerifier{identity: identity()}
		repo := &fakeUserRepo{
			lookups:   []lookupResult{{err: domain.ErrNotFound}, {user: existing}},
			createErr: domain.ErrConflict,
		}
		uc := auth.NewAuthUsecase(verifier, repo)

		got, err := uc.LoginWithGoogle(context.Background(), "id-token")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != existing {
			t.Fatalf("got %v, want %v", got, existing)
		}
	})

	t.Run("email taken by another Google account is a conflict", func(t *testing.T) {
		verifier := &fakeVerifier{identity: identity()}
		repo := &fakeUserRepo{
			lookups:   []lookupResult{{err: domain.ErrNotFound}, {err: domain.ErrNotFound}},
			createErr: domain.ErrConflict,
		}
		uc := auth.NewAuthUsecase(verifier, repo)

		if _, err := uc.LoginWithGoogle(context.Background(), "id-token"); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("got %v, want domain.ErrConflict", err)
		}
	})

	t.Run("create failure is returned", func(t *testing.T) {
		dbErr := errors.New("db down")
		verifier := &fakeVerifier{identity: identity()}
		repo := &fakeUserRepo{
			lookups:   []lookupResult{{err: domain.ErrNotFound}},
			createErr: dbErr,
		}
		uc := auth.NewAuthUsecase(verifier, repo)

		if _, err := uc.LoginWithGoogle(context.Background(), "id-token"); !errors.Is(err, dbErr) {
			t.Fatalf("got %v, want %v", err, dbErr)
		}
	})
}
