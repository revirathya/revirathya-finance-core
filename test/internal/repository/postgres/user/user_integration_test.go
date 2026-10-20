//go:build integration

package user_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"

	// Repository
	rUser "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/user"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}
func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	repo := rUser.NewUserRepositoryImpl(testutil.TestPool)

	t.Run("create stores the email lowercased", func(t *testing.T) {
		id := uuid.NewString()

		user, err := repo.CreateUser(ctx, domain.UserForm{
			FirstName: "Avi",
			LastName:  testutil.Ptr("Leonardi"),
			Email:     "Avi." + id + "@Example.COM",
			GoogleSub: "google-" + id,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := strings.ToLower("Avi." + id + "@Example.COM"); user.Email != want {
			t.Fatalf("email %q, want %q", user.Email, want)
		}
		if user.FirstName != "Avi" || user.LastName == nil || *user.LastName != "Leonardi" {
			t.Fatalf("got %q %v, want Avi Leonardi", user.FirstName, user.LastName)
		}
	})

	t.Run("a user without a telegram binding is found by ID", func(t *testing.T) {
		alice := testutil.NewUser(t)

		got, err := repo.GetUserByID(ctx, alice.ID)

		if err != nil {
			t.Fatalf("got %v, want the user (no telegram binding must not hide it)", err)
		}
		if got.TelegramChatID != nil {
			t.Fatalf("telegram ID %q, want nil", *got.TelegramChatID)
		}
	})

	t.Run("lookup by google sub", func(t *testing.T) {
		id := uuid.NewString()
		alice, err := repo.CreateUser(ctx, domain.UserForm{FirstName: "A", Email: id + "@test.local", GoogleSub: "google-" + id})
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		got, err := repo.GetUserByGoogleSub(ctx, "google-"+id)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.ID != alice.ID {
			t.Fatalf("user %v, want %v", got.ID, alice.ID)
		}

		if _, err := repo.GetUserByGoogleSub(ctx, "google-"+uuid.NewString()); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("unknown sub: got %v, want errDom.ErrNotFound", err)
		}
	})

	t.Run("unknown ID is not found", func(t *testing.T) {
		if _, err := repo.GetUserByID(ctx, uuid.New()); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("got %v, want errDom.ErrNotFound", err)
		}
	})

	t.Run("duplicate email is a conflict, regardless of case", func(t *testing.T) {
		id := uuid.NewString()
		if _, err := repo.CreateUser(ctx, domain.UserForm{FirstName: "A", Email: id + "@test.local", GoogleSub: "google-a-" + id}); err != nil {
			t.Fatalf("first create: %v", err)
		}

		_, err := repo.CreateUser(ctx, domain.UserForm{FirstName: "B", Email: strings.ToUpper(id) + "@TEST.LOCAL", GoogleSub: "google-b-" + id})
		if !errors.Is(err, errDom.ErrConflict) {
			t.Fatalf("got %v, want errDom.ErrConflict", err)
		}
	})

	t.Run("duplicate google sub is a conflict", func(t *testing.T) {
		id := uuid.NewString()
		if _, err := repo.CreateUser(ctx, domain.UserForm{FirstName: "A", Email: "a-" + id + "@test.local", GoogleSub: "google-" + id}); err != nil {
			t.Fatalf("first create: %v", err)
		}

		_, err := repo.CreateUser(ctx, domain.UserForm{FirstName: "B", Email: "b-" + id + "@test.local", GoogleSub: "google-" + id})
		if !errors.Is(err, errDom.ErrConflict) {
			t.Fatalf("got %v, want errDom.ErrConflict", err)
		}
	})

	t.Run("update is a patch: nil keeps a field, a value replaces it, \"\" clears the last name", func(t *testing.T) {
		alice := testutil.NewUser(t)
		if _, err := repo.UpdateUser(ctx, alice.ID, domain.UserPatch{FirstName: testutil.Ptr("Alice"), LastName: testutil.Ptr("Smith")}); err != nil {
			t.Fatalf("first update: %v", err)
		}

		renamed, err := repo.UpdateUser(ctx, alice.ID, domain.UserPatch{FirstName: testutil.Ptr("Alicia")})
		if err != nil {
			t.Fatalf("first-name-only update: %v", err)
		}
		if renamed.FirstName != "Alicia" || renamed.LastName == nil || *renamed.LastName != "Smith" {
			t.Fatalf("got %q %v, want Alicia and last name kept (Smith)", renamed.FirstName, renamed.LastName)
		}

		cleared, err := repo.UpdateUser(ctx, alice.ID, domain.UserPatch{LastName: testutil.Ptr("")})
		if err != nil {
			t.Fatalf("clear last name: %v", err)
		}
		if cleared.FirstName != "Alicia" || cleared.LastName != nil {
			t.Fatalf("got %q %v, want first name kept and last name NULL", cleared.FirstName, cleared.LastName)
		}

		same, err := repo.UpdateUser(ctx, alice.ID, domain.UserPatch{})
		if err != nil || same.FirstName != "Alicia" || same.LastName != nil || same.Email != alice.Email {
			t.Fatalf("empty patch: got %+v, %v; want nothing changed", same, err)
		}
	})

	t.Run("updating an unknown user is not found", func(t *testing.T) {
		if _, err := repo.UpdateUser(ctx, uuid.New(), domain.UserPatch{FirstName: testutil.Ptr("X")}); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("got %v, want errDom.ErrNotFound", err)
		}
	})

	t.Run("database errors are returned, not swallowed", func(t *testing.T) {
		alice := testutil.NewUser(t)
		cancelled, cancel := context.WithCancel(ctx)
		cancel()

		if got, err := repo.GetUserByID(cancelled, alice.ID); err == nil || got != nil {
			t.Fatalf("GetUserByID: got user %v and error %v, want nil user and an error", got, err)
		}
		if got, err := repo.GetUserByGoogleSub(cancelled, "google-x"); err == nil || got != nil {
			t.Fatalf("GetUserByGoogleSub: got user %v and error %v, want nil user and an error", got, err)
		}
		id := uuid.NewString()
		if got, err := repo.CreateUser(cancelled, domain.UserForm{FirstName: "X", Email: id + "@test.local", GoogleSub: id}); err == nil || got != nil {
			t.Fatalf("CreateUser: got user %v and error %v, want nil user and an error", got, err)
		}
		if got, err := repo.UpdateUser(cancelled, alice.ID, domain.UserPatch{FirstName: testutil.Ptr("X")}); err == nil || got != nil {
			t.Fatalf("UpdateUser: got user %v and error %v, want nil user and an error", got, err)
		}
	})
}

// Backstop for input that skipped Validate(): too long values come back as
// ErrInvalidInput, not as an unknown (internal) error.
func TestUserRepositoryLimits(t *testing.T) {
	ctx := context.Background()
	repo := rUser.NewUserRepositoryImpl(testutil.TestPool)
	long := strings.Repeat("a", 101)

	t.Run("too long first name on create is invalid input", func(t *testing.T) {
		id := uuid.NewString()
		_, err := repo.CreateUser(ctx, domain.UserForm{FirstName: long, Email: id + "@test.local", GoogleSub: id})
		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
	})

	t.Run("too long last name on update is invalid input", func(t *testing.T) {
		_, err := repo.UpdateUser(ctx, testutil.NewUser(t).ID, domain.UserPatch{FirstName: testutil.Ptr("Avi"), LastName: &long})
		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
	})
}
