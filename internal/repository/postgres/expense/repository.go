package expense

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	// Generated
	"github.com/revirathya/revirathya-finance-core/gen/sqlc"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
)

type expenseRepositoryImpl struct {
	query *sqlc.Queries
}

func NewExpenseRepositoryImpl(pool *pgxpool.Pool) domain.ExpenseRepository {
	return &expenseRepositoryImpl{
		query: sqlc.New(pool),
	}
}

func (r *expenseRepositoryImpl) CreateExpense(ctx context.Context, form domain.ExpenseForm) (*domain.Expense, error) {
	args := sqlc.CreateExpenseParams{
		UserID:        database.ToPgxUUID(form.UserID),
		CategoryID:    database.ToPgxUUID(form.CategoryID),
		TransactionDt: database.ToPgxDate(form.TransactionDt),
		Description:   form.Description,
		Amount:        database.ToPgxNumeric(form.Amount),
	}

	row, err := r.query.CreateExpense(ctx, args)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "23503":
				// Using other user's category
				return nil, errDom.ErrInvalidInput
			case "22001", "22003":
				// Too long or out of range
				return nil, errDom.ErrInvalidInput
			}
		}
		return nil, err
	}

	result := &domain.Expense{
		ID:            database.FromPgxUUID(row.ID),
		UserID:        database.FromPgxUUID(row.UserID),
		CategoryID:    database.FromPgxUUID(row.CategoryID),
		TransactionDt: database.FromPgxDate(row.TransactionDt),
		Description:   row.Description,
		Amount:        database.FromPgxNumeric(row.Amount),
	}
	return result, nil
}

func (r *expenseRepositoryImpl) GetExpenseByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Expense, error) {
	args := sqlc.GetExpenseByIDParams{
		ID:     database.ToPgxUUID(id),
		UserID: database.ToPgxUUID(userID),
	}

	row, err := r.query.GetExpenseByID(ctx, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errDom.ErrNotFound
		}
		return nil, err
	}

	result := &domain.Expense{
		ID:            database.FromPgxUUID(row.ID),
		UserID:        database.FromPgxUUID(row.UserID),
		CategoryID:    database.FromPgxUUID(row.CategoryID),
		CategoryName:  row.CategoryName,
		TransactionDt: database.FromPgxDate(row.TransactionDt),
		Description:   row.Description,
		Amount:        database.FromPgxNumeric(row.Amount),
	}
	return result, nil
}

func (r *expenseRepositoryImpl) ListExpenses(ctx context.Context, userID uuid.UUID, transactionDt *time.Time) ([]domain.Expense, error) {
	args := sqlc.ListExpensesParams{
		UserID:        database.ToPgxUUID(userID),
		TransactionDt: database.ToNullablePgxDate(transactionDt),
	}

	rows, err := r.query.ListExpenses(ctx, args)
	if err != nil {
		return nil, err
	}

	results := make([]domain.Expense, 0, len(rows))
	for _, row := range rows {
		results = append(results, domain.Expense{
			ID:            database.FromPgxUUID(row.ID),
			CategoryID:    database.FromPgxUUID(row.CategoryID),
			UserID:        database.FromPgxUUID(row.UserID),
			CategoryName:  row.CategoryName,
			TransactionDt: database.FromPgxDate(row.TransactionDt),
			Description:   row.Description,
			Amount:        database.FromPgxNumeric(row.Amount),
		})
	}
	return results, nil
}

func (r *expenseRepositoryImpl) UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.ExpensePatch) (*domain.Expense, error) {
	args := sqlc.UpdateExpenseParams{
		CategoryID:    database.ToPgxUUID(patch.CategoryID),
		TransactionDt: database.ToPgxDate(patch.TransactionDt),
		Description:   patch.Description,
		Amount:        database.ToPgxNumeric(patch.Amount),
		ID:            database.ToPgxUUID(id),
		UserID:        database.ToPgxUUID(userID),
	}

	row, err := r.query.UpdateExpense(ctx, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not found to-be-updated expense
			return nil, errDom.ErrNotFound
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "23503":
				// Using other user's category
				return nil, errDom.ErrInvalidInput
			case "22001", "22003":
				// Too long or out of range
				return nil, errDom.ErrInvalidInput
			}
		}
		return nil, err
	}

	result := &domain.Expense{
		ID:            database.FromPgxUUID(row.ID),
		UserID:        database.FromPgxUUID(row.UserID),
		CategoryID:    database.FromPgxUUID(row.CategoryID),
		TransactionDt: database.FromPgxDate(row.TransactionDt),
		Description:   row.Description,
		Amount:        database.FromPgxNumeric(row.Amount),
	}
	return result, nil
}

func (r *expenseRepositoryImpl) DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := sqlc.DeleteExpenseParams{
		ID:     database.ToPgxUUID(id),
		UserID: database.ToPgxUUID(userID),
	}

	rowCnt, err := r.query.DeleteExpense(ctx, args)
	if err != nil {
		return err
	}
	if rowCnt == 0 {
		return errDom.ErrNotFound
	}
	return nil
}

// Sanity Check
var _ domain.ExpenseRepository = (&expenseRepositoryImpl{})
