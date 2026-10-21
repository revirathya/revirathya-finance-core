package user_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/user"
)

// Fake
type fakeUserRepo struct {
	userByID        *domain.User
	userByGoogleSub *domain.User
	createResp      *domain.User
	updateResp      *domain.User
	err             error

	gotGetID          uuid.UUID
	gotGoogleSub      string
	gotCreateForm     domain.UserForm
	gotUpdateID       uuid.UUID
	gotUpdateFirst    *string
	gotUpdateLastName *string
}

func (f *fakeUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	f.gotGetID = id
	return f.userByID, f.err
}

func (f *fakeUserRepo) GetUserByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	f.gotGoogleSub = googleSub
	return f.userByGoogleSub, f.err
}

func (f *fakeUserRepo) CreateUser(ctx context.Context, form domain.UserForm) (*domain.User, error) {
	f.gotCreateForm = form
	return f.createResp, f.err
}

func (f *fakeUserRepo) UpdateUser(ctx context.Context, id uuid.UUID, patch domain.UserPatch) (*domain.User, error) {
	f.gotUpdateID = id
	f.gotUpdateFirst = patch.FirstName
	f.gotUpdateLastName = patch.LastName
	return f.updateResp, f.err
}

// Test
func TestGetUserByID(t *testing.T) {
	id := uuid.New()
	want := &domain.User{ID: id, FirstName: "Avi", Email: "avi@example.com"}

	t.Run("returns user", func(t *testing.T) {
		repo := &fakeUserRepo{userByID: want}
		uc := user.NewUserUsecase(repo)

		got, err := uc.GetUserByID(context.Background(), id)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotGetID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotGetID, id)
		}
	})

	t.Run("propagates not found", func(t *testing.T) {
		repo := &fakeUserRepo{err: errDom.ErrNotFound}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.GetUserByID(context.Background(), id); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("got %v, want errDom.ErrNotFound", err)
		}
	})
}

func TestUpdateUser(t *testing.T) {
	id := uuid.New()
	lastName := "Leonardi"
	want := &domain.User{ID: id, FirstName: "Avi", LastName: &lastName, Email: "avi@example.com"}

	t.Run("updates and returns user", func(t *testing.T) {
		repo := &fakeUserRepo{updateResp: want}
		uc := user.NewUserUsecase(repo)

		got, err := uc.UpdateUser(context.Background(), id, domain.UserPatch{FirstName: ptr("Avi"), LastName: &lastName})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotUpdateID != id || repo.gotUpdateFirst == nil || *repo.gotUpdateFirst != "Avi" || repo.gotUpdateLastName == nil || *repo.gotUpdateLastName != lastName {
			t.Fatalf("repo received (%v, %v, %v), want (%v, %q, %q)",
				repo.gotUpdateID, repo.gotUpdateFirst, repo.gotUpdateLastName, id, "Avi", lastName)
		}
	})

	t.Run("omitted fields reach the repository as nil (keep)", func(t *testing.T) {
		repo := &fakeUserRepo{updateResp: want}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.UpdateUser(context.Background(), id, domain.UserPatch{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotUpdateFirst != nil || repo.gotUpdateLastName != nil {
			t.Fatalf("repo received %v %v, want both nil", repo.gotUpdateFirst, repo.gotUpdateLastName)
		}
	})

	t.Run("propagates not found", func(t *testing.T) {
		repo := &fakeUserRepo{err: errDom.ErrNotFound}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.UpdateUser(context.Background(), id, domain.UserPatch{FirstName: ptr("Avi"), LastName: &lastName}); !errors.Is(err, errDom.ErrNotFound) {
			t.Fatalf("got %v, want errDom.ErrNotFound", err)
		}
	})
}

func ptr[T any](v T) *T { return &v }
