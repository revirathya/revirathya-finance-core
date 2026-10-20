package category_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/domain/constant"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/category"
)

// Fake
type fakeCategoryRepo struct {
	categories []domain.Category

	gotListUserID uuid.UUID
	gotListQuery  domain.CategoryListQuery
	categoryByID  *domain.Category
	createResp    *domain.Category
	updateResp    *domain.Category
	err           error

	gotCreateInput domain.CategoryForm
	gotUpdateID    uuid.UUID
	gotUpdatePatch domain.CategoryPatch
	gotDeleteID    uuid.UUID
}

func (f *fakeCategoryRepo) ListCategories(ctx context.Context, userID uuid.UUID, query domain.CategoryListQuery) ([]domain.Category, error) {
	f.gotListUserID = userID
	f.gotListQuery = query
	return f.categories, f.err
}

func (f *fakeCategoryRepo) GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error) {
	return f.categoryByID, f.err
}

func (f *fakeCategoryRepo) CreateCategory(ctx context.Context, userID uuid.UUID, input domain.CategoryForm) (*domain.Category, error) {
	f.gotCreateInput = input
	return f.createResp, f.err
}

func (f *fakeCategoryRepo) UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.CategoryPatch) (*domain.Category, error) {
	f.gotUpdateID = id
	f.gotUpdatePatch = patch
	return f.updateResp, f.err
}

func (f *fakeCategoryRepo) DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	f.gotDeleteID = id
	return f.err
}

// Test
func TestCreateCategory(t *testing.T) {
	want := &domain.Category{ID: uuid.New(), CategoryName: "Food"}

	t.Run("creates and returns category", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{createResp: want}
		uc := category.NewCategoryUsecase(repo)

		got, err := uc.CreateCategory(context.Background(), userID, "Food")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotCreateInput.CategoryName != "Food" {
			t.Fatalf("repo received category_name %q, want %q", repo.gotCreateInput.CategoryName, "Food")
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{err: errors.New("db down")}
		uc := category.NewCategoryUsecase(repo)

		if _, err := uc.CreateCategory(context.Background(), userID, "Food"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetCategoryByID(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	want := &domain.Category{ID: id, CategoryName: "Food"}

	t.Run("returns category", func(t *testing.T) {
		repo := &fakeCategoryRepo{categoryByID: want}
		uc := category.NewCategoryUsecase(repo)

		got, err := uc.GetCategoryByID(context.Background(), id, userID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeCategoryRepo{err: errors.New("not found")}
		uc := category.NewCategoryUsecase(repo)

		if _, err := uc.GetCategoryByID(context.Background(), id, userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestListCategories(t *testing.T) {
	userID := uuid.New()
	names := func(ns ...string) []domain.Category {
		out := make([]domain.Category, len(ns))
		for i, n := range ns {
			out[i] = domain.Category{ID: uuid.New(), UserID: userID, CategoryName: n}
		}
		return out
	}

	t.Run("first page asks for page size + 1 with no cursor", func(t *testing.T) {
		repo := &fakeCategoryRepo{categories: names("Bills", "Food")}
		uc := category.NewCategoryUsecase(repo)

		page, err := uc.ListCategories(context.Background(), userID, 10, "")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotListUserID != userID || repo.gotListQuery.After != nil || repo.gotListQuery.Limit != 11 {
			t.Fatalf("repo received %v %+v, want the user, no cursor, limit 11", repo.gotListUserID, repo.gotListQuery)
		}
		if len(page.Categories) != 2 || page.NextCursor != nil {
			t.Fatalf("got %d categories, cursor %v; want 2 and no next page", len(page.Categories), page.NextCursor)
		}
	})

	t.Run("page size defaults and is capped", func(t *testing.T) {
		for _, tt := range []struct{ in, wantLimit int }{
			{0, constant.DefaultPageSize + 1},
			{constant.MaxPageSize + 1, constant.MaxPageSize + 1},
			{1_000_000, constant.MaxPageSize + 1},
		} {
			repo := &fakeCategoryRepo{}
			if _, err := category.NewCategoryUsecase(repo).ListCategories(context.Background(), userID, tt.in, ""); err != nil {
				t.Fatalf("page size %d: %v", tt.in, err)
			}
			if repo.gotListQuery.Limit != tt.wantLimit {
				t.Fatalf("page size %d: repo limit %d, want %d", tt.in, repo.gotListQuery.Limit, tt.wantLimit)
			}
		}
	})

	t.Run("an extra row means a next page whose cursor is the last returned name", func(t *testing.T) {
		repo := &fakeCategoryRepo{categories: names("Bills", "Food", "Transport")}

		page, err := category.NewCategoryUsecase(repo).ListCategories(context.Background(), userID, 2, "")

		if err != nil || len(page.Categories) != 2 || page.NextCursor == nil || page.NextCursor.CategoryName != "Food" {
			t.Fatalf("got %+v, %v; want Bills, Food and a cursor at Food", page, err)
		}
	})

	t.Run("the default and capped sizes are the ones used to cut the page", func(t *testing.T) {
		many := make([]string, constant.MaxPageSize+5)
		for i := range many {
			many[i] = fmt.Sprintf("c%03d", i)
		}
		for _, tt := range []struct{ in, want int }{{0, constant.DefaultPageSize}, {1_000_000, constant.MaxPageSize}} {
			rows := names(many[:tt.want+1]...) // what the repository returns for limit want+1
			page, err := category.NewCategoryUsecase(&fakeCategoryRepo{categories: rows}).ListCategories(context.Background(), userID, tt.in, "")
			if err != nil || len(page.Categories) != tt.want || page.NextCursor == nil || page.NextCursor.CategoryName != rows[tt.want-1].CategoryName {
				t.Fatalf("page size %d: got %d categories, cursor %v, %v; want %d and a cursor at the last one", tt.in, len(page.Categories), page.NextCursor, err, tt.want)
			}
		}
	})

	t.Run("exactly page size rows is the last page", func(t *testing.T) {
		page, err := category.NewCategoryUsecase(&fakeCategoryRepo{categories: names("Bills", "Food")}).ListCategories(context.Background(), userID, 2, "")
		if err != nil || page.NextCursor != nil {
			t.Fatalf("got cursor %v, %v; want no next page", page.NextCursor, err)
		}
	})

	t.Run("page token is decoded into the cursor", func(t *testing.T) {
		repo := &fakeCategoryRepo{}
		token := domain.CategoryCursor{CategoryName: "Food"}.Encode()

		if _, err := category.NewCategoryUsecase(repo).ListCategories(context.Background(), userID, 0, token); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotListQuery.After == nil || repo.gotListQuery.After.CategoryName != "Food" {
			t.Fatalf("repo cursor %v, want Food", repo.gotListQuery.After)
		}
	})

	t.Run("invalid input is a ValidationError naming the field, before the repository", func(t *testing.T) {
		for field, call := range map[string]func(uc *category.CategoryUsecase) error{
			"page_size": func(uc *category.CategoryUsecase) error {
				_, err := uc.ListCategories(context.Background(), userID, -1, "")
				return err
			},
			"page_token": func(uc *category.CategoryUsecase) error {
				_, err := uc.ListCategories(context.Background(), userID, 0, "garbage!")
				return err
			},
		} {
			repo := &fakeCategoryRepo{}
			err := call(category.NewCategoryUsecase(repo))

			var vErr *errDom.ValidationError
			if !errors.As(err, &vErr) || vErr.Field != field || vErr.Reason == "" {
				t.Fatalf("%s: got %#v, want a ValidationError with Field %q and a reason", field, err, field)
			}
			if repo.gotListUserID != uuid.Nil {
				t.Fatalf("%s: repository was called", field)
			}
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		if _, err := category.NewCategoryUsecase(&fakeCategoryRepo{err: errors.New("db down")}).ListCategories(context.Background(), userID, 0, ""); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestUpdateCategory(t *testing.T) {
	id := uuid.New()
	want := &domain.Category{ID: id, CategoryName: "Groceries"}

	t.Run("updates and returns category", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{updateResp: want}
		uc := category.NewCategoryUsecase(repo)

		got, err := uc.UpdateCategory(context.Background(), id, userID, domain.CategoryPatch{CategoryName: ptr("Groceries")})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotUpdateID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotUpdateID, id)
		}
		if repo.gotUpdatePatch.CategoryName == nil || *repo.gotUpdatePatch.CategoryName != "Groceries" {
			t.Fatalf("repo received category_name %v, want %q", repo.gotUpdatePatch.CategoryName, "Groceries")
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{err: errors.New("not found")}
		uc := category.NewCategoryUsecase(repo)

		if _, err := uc.UpdateCategory(context.Background(), id, userID, domain.CategoryPatch{CategoryName: ptr("Groceries")}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDeleteCategory(t *testing.T) {
	id := uuid.New()

	t.Run("deletes category", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{}
		uc := category.NewCategoryUsecase(repo)

		if err := uc.DeleteCategory(context.Background(), id, userID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotDeleteID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotDeleteID, id)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		userID := uuid.New()
		repo := &fakeCategoryRepo{err: errors.New("fk violation")}
		uc := category.NewCategoryUsecase(repo)

		if err := uc.DeleteCategory(context.Background(), id, userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func ptr[T any](v T) *T { return &v }
