package user_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
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
	gotUpdateFirst    string
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

func (f *fakeUserRepo) UpdateUser(ctx context.Context, id uuid.UUID, firstName string, lastName *string) (*domain.User, error) {
	f.gotUpdateID = id
	f.gotUpdateFirst = firstName
	f.gotUpdateLastName = lastName
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
		repo := &fakeUserRepo{err: domain.ErrNotFound}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.GetUserByID(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("got %v, want domain.ErrNotFound", err)
		}
	})
}

func TestGetUserByGoogleSub(t *testing.T) {
	const sub = "google-sub-123"
	want := &domain.User{ID: uuid.New(), FirstName: "Avi", Email: "avi@example.com"}

	t.Run("returns user", func(t *testing.T) {
		repo := &fakeUserRepo{userByGoogleSub: want}
		uc := user.NewUserUsecase(repo)

		got, err := uc.GetUserByGoogleSub(context.Background(), sub)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotGoogleSub != sub {
			t.Fatalf("repo received google_sub %q, want %q", repo.gotGoogleSub, sub)
		}
	})

	t.Run("propagates not found", func(t *testing.T) {
		repo := &fakeUserRepo{err: domain.ErrNotFound}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.GetUserByGoogleSub(context.Background(), sub); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("got %v, want domain.ErrNotFound", err)
		}
	})
}

func TestCreateUser(t *testing.T) {
	lastName := "Leonardi"
	form := domain.UserForm{
		FirstName: "Avi",
		LastName:  &lastName,
		Email:     "avi@example.com",
		GoogleSub: "google-sub-123",
	}
	want := &domain.User{ID: uuid.New(), FirstName: "Avi", LastName: &lastName, Email: "avi@example.com"}

	t.Run("creates and returns user", func(t *testing.T) {
		repo := &fakeUserRepo{createResp: want}
		uc := user.NewUserUsecase(repo)

		got, err := uc.CreateUser(context.Background(), form)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if !reflect.DeepEqual(repo.gotCreateForm, form) {
			t.Fatalf("repo received form %+v, want %+v", repo.gotCreateForm, form)
		}
	})

	t.Run("propagates conflict", func(t *testing.T) {
		repo := &fakeUserRepo{err: domain.ErrConflict}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.CreateUser(context.Background(), form); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("got %v, want domain.ErrConflict", err)
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

		got, err := uc.UpdateUser(context.Background(), id, "Avi", &lastName)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotUpdateID != id || repo.gotUpdateFirst != "Avi" || repo.gotUpdateLastName != &lastName {
			t.Fatalf("repo received (%v, %q, %v), want (%v, %q, %v)",
				repo.gotUpdateID, repo.gotUpdateFirst, repo.gotUpdateLastName, id, "Avi", &lastName)
		}
	})

	t.Run("clears last name with nil", func(t *testing.T) {
		repo := &fakeUserRepo{updateResp: want}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.UpdateUser(context.Background(), id, "Avi", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotUpdateLastName != nil {
			t.Fatalf("repo received last_name %v, want nil", *repo.gotUpdateLastName)
		}
	})

	t.Run("propagates not found", func(t *testing.T) {
		repo := &fakeUserRepo{err: domain.ErrNotFound}
		uc := user.NewUserUsecase(repo)

		if _, err := uc.UpdateUser(context.Background(), id, "Avi", &lastName); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("got %v, want domain.ErrNotFound", err)
		}
	})
}
