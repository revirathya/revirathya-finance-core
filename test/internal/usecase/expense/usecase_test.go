package expense_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	"github.com/shopspring/decimal"
)

// Fake
type fakeExpenseRepo struct {
	createResp *domain.Expense
	getResp    *domain.Expense
	listResp   []domain.Expense
	updateResp *domain.Expense
	err        error

	gotCreateInput domain.ExpenseForm
	gotGetID       uuid.UUID
	gotListUserID  uuid.UUID
	gotListQuery   domain.ExpenseListQuery

	gotUpdateID    uuid.UUID
	gotUpdatePatch domain.ExpensePatch

	gotDeleteID uuid.UUID
}

func (f *fakeExpenseRepo) CreateExpense(ctx context.Context, input domain.ExpenseForm) (*domain.Expense, error) {
	f.gotCreateInput = input
	return f.createResp, f.err
}

func (f *fakeExpenseRepo) GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Expense, error) {
	f.gotGetID = id
	return f.getResp, f.err
}

func (f *fakeExpenseRepo) ListExpenses(ctx context.Context, userID uuid.UUID, query domain.ExpenseListQuery) ([]domain.Expense, error) {
	f.gotListUserID = userID
	f.gotListQuery = query
	return f.listResp, f.err
}

func (f *fakeExpenseRepo) UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.ExpensePatch) (*domain.Expense, error) {
	f.gotUpdateID = id
	f.gotUpdatePatch = patch
	return f.updateResp, f.err
}

func (f *fakeExpenseRepo) DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	f.gotDeleteID = id
	return f.err
}

// Tests
func TestCreateExpense(t *testing.T) {
	userID, categoryID := uuid.New(), uuid.New()
	dt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	amount := decimal.NewFromFloat(42.50)
	description := "Lunch"
	want := &domain.Expense{ID: uuid.New(), Description: description}

	t.Run("creates and returns expense", func(t *testing.T) {
		repo := &fakeExpenseRepo{createResp: want}
		uc := expense.NewExpenseUsecase(repo)

		got, err := uc.CreateExpense(context.Background(), userID, categoryID, dt, &description, amount)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}

		wantInput := domain.ExpenseForm{
			UserID:        userID,
			CategoryID:    categoryID,
			TransactionDt: dt,
			Description:   &description,
			Amount:        amount,
		}
		if !reflect.DeepEqual(repo.gotCreateInput, wantInput) {
			t.Fatalf("repo received %v, want %v", repo.gotCreateInput, wantInput)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeExpenseRepo{err: errors.New("db down")}
		uc := expense.NewExpenseUsecase(repo)

		if _, err := uc.CreateExpense(context.Background(), userID, categoryID, dt, &description, amount); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetExpenseByID(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	description := "Coffee"
	want := &domain.Expense{ID: id, Description: description}

	t.Run("returns expense", func(t *testing.T) {
		repo := &fakeExpenseRepo{getResp: want}
		uc := expense.NewExpenseUsecase(repo)

		got, err := uc.GetExpenseByID(context.Background(), id, userID)

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

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeExpenseRepo{err: errors.New("not found")}
		uc := expense.NewExpenseUsecase(repo)

		if _, err := uc.GetExpenseByID(context.Background(), id, userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestListExpenses(t *testing.T) {
	userID := uuid.New()
	dt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	// expenses returns n expenses, newest first, as the repository would.
	expenses := func(n int) []domain.Expense {
		out := make([]domain.Expense, n)
		for i := range out {
			out[i] = domain.Expense{ID: uuid.New(), UserID: userID, TransactionDt: dt.AddDate(0, 0, -i)}
		}
		return out
	}

	t.Run("first page passes the user, filter and page size + 1, with no cursor", func(t *testing.T) {
		repo := &fakeExpenseRepo{listResp: expenses(3)}
		uc := expense.NewExpenseUsecase(repo)

		page, err := uc.ListExpenses(context.Background(), userID, &dt, 10, "")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		q := repo.gotListQuery
		if repo.gotListUserID != userID || q.TransactionDt == nil || !q.TransactionDt.Equal(dt) || q.After != nil || q.Limit != 11 {
			t.Fatalf("repo received user %v, query %+v; want the user, the date filter, no cursor and limit 11", repo.gotListUserID, q)
		}
		if len(page.Expenses) != 3 || page.NextCursor != nil {
			t.Fatalf("got %d expenses and cursor %v, want all 3 and no next page", len(page.Expenses), page.NextCursor)
		}
	})

	t.Run("page size defaults and is capped", func(t *testing.T) {
		for _, tt := range []struct{ in, wantLimit int }{
			{0, domain.DefaultPageSize + 1},
			{domain.MaxPageSize, domain.MaxPageSize + 1},
			{domain.MaxPageSize + 1, domain.MaxPageSize + 1},
			{1_000_000, domain.MaxPageSize + 1},
		} {
			repo := &fakeExpenseRepo{}
			uc := expense.NewExpenseUsecase(repo)

			if _, err := uc.ListExpenses(context.Background(), userID, nil, tt.in, ""); err != nil {
				t.Fatalf("page size %d: unexpected error: %v", tt.in, err)
			}
			if repo.gotListQuery.Limit != tt.wantLimit {
				t.Fatalf("page size %d: repo limit %d, want %d", tt.in, repo.gotListQuery.Limit, tt.wantLimit)
			}
		}
	})

	t.Run("an extra row means a next page: it is dropped and the last row becomes the cursor", func(t *testing.T) {
		rows := expenses(3)
		repo := &fakeExpenseRepo{listResp: rows}
		uc := expense.NewExpenseUsecase(repo)

		page, err := uc.ListExpenses(context.Background(), userID, nil, 2, "")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Expenses) != 2 || page.Expenses[1].ID != rows[1].ID {
			t.Fatalf("got %d expenses, want the first 2", len(page.Expenses))
		}
		want := domain.ExpenseCursor{TransactionDt: rows[1].TransactionDt, ID: rows[1].ID}
		if page.NextCursor == nil || *page.NextCursor != want {
			t.Fatalf("next cursor %v, want %v (the last returned row)", page.NextCursor, want)
		}
	})

	t.Run("exactly page size rows is the last page", func(t *testing.T) {
		repo := &fakeExpenseRepo{listResp: expenses(2)}
		uc := expense.NewExpenseUsecase(repo)

		page, err := uc.ListExpenses(context.Background(), userID, nil, 2, "")

		if err != nil || len(page.Expenses) != 2 || page.NextCursor != nil {
			t.Fatalf("got %d expenses, cursor %v, err %v; want 2 and no next page", len(page.Expenses), page.NextCursor, err)
		}
	})

	t.Run("empty result is an empty last page", func(t *testing.T) {
		uc := expense.NewExpenseUsecase(&fakeExpenseRepo{})

		page, err := uc.ListExpenses(context.Background(), userID, nil, 0, "")

		if err != nil || len(page.Expenses) != 0 || page.NextCursor != nil {
			t.Fatalf("got %v, %v; want an empty page without cursor", page, err)
		}
	})

	t.Run("page token is decoded into the cursor", func(t *testing.T) {
		cursor := domain.ExpenseCursor{TransactionDt: dt, ID: uuid.New()}
		repo := &fakeExpenseRepo{}
		uc := expense.NewExpenseUsecase(repo)

		if _, err := uc.ListExpenses(context.Background(), userID, nil, 0, cursor.Encode()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotListQuery.After == nil || *repo.gotListQuery.After != cursor {
			t.Fatalf("repo cursor %v, want %v", repo.gotListQuery.After, cursor)
		}
	})

	t.Run("invalid input is rejected before the repository", func(t *testing.T) {
		for name, call := range map[string]func(uc *expense.ExpenseUsecase) error{
			"page_size": func(uc *expense.ExpenseUsecase) error {
				_, err := uc.ListExpenses(context.Background(), userID, nil, -1, "")
				return err
			},
			"page_token": func(uc *expense.ExpenseUsecase) error {
				_, err := uc.ListExpenses(context.Background(), userID, nil, 0, "garbage!")
				return err
			},
		} {
			repo := &fakeExpenseRepo{}
			err := call(expense.NewExpenseUsecase(repo))

			var vErr *errDom.ValidationError
			if !errors.As(err, &vErr) || vErr.Field != name {
				t.Fatalf("%s: got %v, want a ValidationError for %s", name, err, name)
			}
			if repo.gotListUserID != uuid.Nil {
				t.Fatalf("%s: repository was called", name)
			}
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		uc := expense.NewExpenseUsecase(&fakeExpenseRepo{err: errors.New("db down")})

		if _, err := uc.ListExpenses(context.Background(), userID, nil, 0, ""); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestUpdateExpense(t *testing.T) {
	id, userID, categoryID := uuid.New(), uuid.New(), uuid.New()
	dt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	amount := decimal.NewFromFloat(42.50)
	description := "Dinner"
	want := &domain.Expense{ID: id, Amount: amount}

	t.Run("updates and returns expense", func(t *testing.T) {
		repo := &fakeExpenseRepo{updateResp: want}
		uc := expense.NewExpenseUsecase(repo)

		got, err := uc.UpdateExpense(context.Background(), id, userID, categoryID, dt, &description, amount)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotUpdateID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotUpdateID, id)
		}

		wantPatch := domain.ExpensePatch{
			CategoryID:    categoryID,
			TransactionDt: dt,
			Description:   &description,
			Amount:        amount,
		}
		if !reflect.DeepEqual(repo.gotUpdatePatch, wantPatch) {
			t.Fatalf("repo received patch %v, want %v", repo.gotUpdatePatch, wantPatch)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeExpenseRepo{err: errors.New("not found")}
		uc := expense.NewExpenseUsecase(repo)

		if _, err := uc.UpdateExpense(context.Background(), id, userID, categoryID, dt, &description, amount); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDeleteExpense(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()

	t.Run("deletes expense", func(t *testing.T) {
		repo := &fakeExpenseRepo{}
		uc := expense.NewExpenseUsecase(repo)

		if err := uc.DeleteExpense(context.Background(), id, userID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotDeleteID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotDeleteID, id)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeExpenseRepo{err: errors.New("fk violation")}
		uc := expense.NewExpenseUsecase(repo)

		if err := uc.DeleteExpense(context.Background(), id, userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
