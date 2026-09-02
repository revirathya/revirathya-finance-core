package expense_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
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
	gotListDt      *time.Time

	gotUpdateID    uuid.UUID
	gotUpdatePatch domain.ExpenseForm

	gotDeleteID uuid.UUID
}

func (f *fakeExpenseRepo) CreateExpense(ctx context.Context, input domain.ExpenseForm) (*domain.Expense, error) {
	f.gotCreateInput = input
	return f.createResp, f.err
}

func (f *fakeExpenseRepo) GetExpenseByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	f.gotGetID = id
	return f.getResp, f.err
}

func (f *fakeExpenseRepo) ListExpenses(ctx context.Context, userID uuid.UUID, transactionDt *time.Time) ([]domain.Expense, error) {
	f.gotListUserID = userID
	f.gotListDt = transactionDt
	return f.listResp, f.err
}

func (f *fakeExpenseRepo) UpdateExpense(ctx context.Context, id uuid.UUID, patch domain.ExpenseForm) (*domain.Expense, error) {
	f.gotUpdateID = id
	f.gotUpdatePatch = patch
	return f.updateResp, f.err
}

func (f *fakeExpenseRepo) DeleteExpense(ctx context.Context, id uuid.UUID) error {
	f.gotDeleteID = id
	return f.err
}

// Tests
func TestCreateExpense(t *testing.T) {
	userID, categoryID := uuid.New(), uuid.New()
	dt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	amount := decimal.NewFromFloat(42.50)
	description := "Lunch"
	want := &domain.Expense{ID: uuid.New(), Description: &description}

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
	description := "Coffee"
	want := &domain.Expense{ID: id, Description: &description}

	t.Run("returns expense", func(t *testing.T) {
		repo := &fakeExpenseRepo{getResp: want}
		uc := expense.NewExpenseUsecase(repo)

		got, err := uc.GetExpenseByID(context.Background(), id)

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

		if _, err := uc.GetExpenseByID(context.Background(), id); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestListExpenses(t *testing.T) {
	userID := uuid.New()
	dt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	want := []domain.Expense{{ID: uuid.New(), UserID: userID}}

	t.Run("returns expenses without date filter", func(t *testing.T) {
		repo := &fakeExpenseRepo{listResp: want}
		uc := expense.NewExpenseUsecase(repo)

		got, err := uc.ListExpenses(context.Background(), userID, nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if repo.gotListUserID != userID {
			t.Fatalf("repo received userID %v, want %v", repo.gotListUserID, userID)
		}
		if repo.gotListDt != nil {
			t.Fatalf("repo received date filter %v, want nil", repo.gotListDt)
		}
	})

	t.Run("passes date filter through", func(t *testing.T) {
		repo := &fakeExpenseRepo{listResp: want}
		uc := expense.NewExpenseUsecase(repo)

		if _, err := uc.ListExpenses(context.Background(), userID, &dt); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotListDt == nil || !repo.gotListDt.Equal(dt) {
			t.Fatalf("repo received date filter %v, want %v", repo.gotListDt, dt)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeExpenseRepo{err: errors.New("db down")}
		uc := expense.NewExpenseUsecase(repo)

		if _, err := uc.ListExpenses(context.Background(), userID, nil); err == nil {
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

		wantPatch := domain.ExpenseForm{
			UserID:        userID,
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

	t.Run("deletes expense", func(t *testing.T) {
		repo := &fakeExpenseRepo{}
		uc := expense.NewExpenseUsecase(repo)

		if err := uc.DeleteExpense(context.Background(), id); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotDeleteID != id {
			t.Fatalf("repo received id %v, want %v", repo.gotDeleteID, id)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeExpenseRepo{err: errors.New("fk violation")}
		uc := expense.NewExpenseUsecase(repo)

		if err := uc.DeleteExpense(context.Background(), id); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
