package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
	sqlcgen "github.com/revirathya/revirathya-finance-core/internal/repository/sqlc/gen"
)

type expenseRepositoryImpl struct {
	query *sqlcgen.Queries
}

func NewExpenseRepositoryImpl(pool *pgxpool.Pool) domain.ExpenseRepository {
	return &expenseRepositoryImpl{
		query: sqlcgen.New(pool),
	}
}

func (r *expenseRepositoryImpl) CreateExpense(ctx context.Context, input domain.ExpenseForm) (*domain.Expense, error) {
	args := sqlcgen.CreateExpenseParams{
		UserID:        database.ToPgxUUID(input.UserID),
		CategoryID:    database.ToPgxUUID(input.CategoryID),
		TransactionDt: database.ToPgxDate(input.TransactionDt),
		Description:   input.Description,
		Amount:        database.ToPgxNumeric(input.Amount),
	}

	row, err := r.query.CreateExpense(ctx, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr)) && (pgErr.Code == "23503") {
			// Using other user's category
			return nil, domain.ErrInvalidInput
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

func (r *expenseRepositoryImpl) GetExpenseByID(ctx context.Context, id uuid.UUID) (*domain.Expense, error) {
	row, err := r.query.GetExpenseByID(ctx, database.ToPgxUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
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
	args := sqlcgen.ListExpensesParams{
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

func (r *expenseRepositoryImpl) UpdateExpense(ctx context.Context, id uuid.UUID, patch domain.ExpenseForm) (*domain.Expense, error) {
	args := sqlcgen.UpdateExpenseParams{
		CategoryID:    database.ToPgxUUID(patch.CategoryID),
		TransactionDt: database.ToPgxDate(patch.TransactionDt),
		Description:   patch.Description,
		Amount:        database.ToPgxNumeric(patch.Amount),
		ID:            database.ToPgxUUID(id),
	}

	row, err := r.query.UpdateExpense(ctx, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr)) && (pgErr.Code == "23503") {
			// Using other user's category
			return nil, domain.ErrInvalidInput
		}
		if errors.Is(err, pgx.ErrNoRows) {
			// Not found to-be-updated expense
			return nil, domain.ErrNotFound
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

func (r *expenseRepositoryImpl) DeleteExpense(ctx context.Context, id uuid.UUID) error {
	if err := r.query.DeleteExpense(ctx, database.ToPgxUUID(id)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

// Sanity Check
var _ domain.ExpenseRepository = (&expenseRepositoryImpl{})
