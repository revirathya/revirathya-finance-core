package user_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/user"
)

func TestUserValidation(t *testing.T) {
	long := strings.Repeat("a", 101)

	t.Run("create rejects invalid input before the repository", func(t *testing.T) {
		repo := &fakeUserRepo{}
		uc := user.NewUserUsecase(repo)

		_, err := uc.CreateUser(context.Background(), "  ", nil, "avi@example.com", "123")

		if !errors.Is(err, errDom.ErrInvalidInput) {
			t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
		}
		if repo.gotCreateForm.Email != "" {
			t.Fatal("repository was called")
		}
	})

	blankFirst := "  "
	for name, patch := range map[string]domain.UserPatch{"blank first name": {FirstName: &blankFirst}, "too long last name": {LastName: &long}} {
		t.Run("update rejects "+name+" before the repository", func(t *testing.T) {
			repo := &fakeUserRepo{}
			uc := user.NewUserUsecase(repo)

			_, err := uc.UpdateUser(context.Background(), uuid.New(), patch)

			if !errors.Is(err, errDom.ErrInvalidInput) {
				t.Fatalf("got %v, want errDom.ErrInvalidInput", err)
			}
			if repo.gotUpdateID != uuid.Nil {
				t.Fatal("repository was called")
			}
		})
	}

	t.Run("update passes a blank last name to the repository as \"\" (clear), not nil (keep)", func(t *testing.T) {
		repo := &fakeUserRepo{}
		uc := user.NewUserUsecase(repo)
		blank := "   "

		if _, err := uc.UpdateUser(context.Background(), uuid.New(), domain.UserPatch{LastName: &blank}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotUpdateLastName == nil || *repo.gotUpdateLastName != "" {
			t.Fatalf("repository received last name %v, want a pointer to \"\"", repo.gotUpdateLastName)
		}
	})

	t.Run("update passes trimmed names to the repository", func(t *testing.T) {
		repo := &fakeUserRepo{}
		uc := user.NewUserUsecase(repo)
		lastName := " Leonardi "

		if _, err := uc.UpdateUser(context.Background(), uuid.New(), domain.UserPatch{FirstName: ptr(" Avi "), LastName: &lastName}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotUpdateFirst == nil || *repo.gotUpdateFirst != "Avi" || repo.gotUpdateLastName == nil || *repo.gotUpdateLastName != "Leonardi" {
			t.Fatalf("repository received %v %v, want trimmed names", repo.gotUpdateFirst, repo.gotUpdateLastName)
		}
	})
}
