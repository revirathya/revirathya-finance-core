package category

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	// Generated
	"github.com/revirathya/revirathya-finance-core/gen/sqlc"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
)

type categoryRepositoryImpl struct {
	query *sqlc.Queries
}

func NewCategoryRepositoryImpl(pool *pgxpool.Pool) domain.CategoryRepository {
	return &categoryRepositoryImpl{
		query: sqlc.New(pool),
	}
}

func (r *categoryRepositoryImpl) CreateCategory(ctx context.Context, userID uuid.UUID, input domain.CategoryForm) (*domain.Category, error) {
	args := sqlc.CreateCategoryParams{
		UserID:       database.ToPgxUUID(userID),
		CategoryName: input.CategoryName,
	}

	row, err := r.query.CreateCategory(ctx, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr)) && (pgErr.Code == "23505") {
			// Duplicate names for current user
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	result := &domain.Category{
		ID:           database.FromPgxUUID(row.ID),
		UserID:       database.FromPgxUUID(row.UserID),
		CategoryName: row.CategoryName,
	}
	return result, nil
}

func (r *categoryRepositoryImpl) GetCategoryByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Category, error) {
	args := sqlc.GetCategoryByIDParams{
		ID:     database.ToPgxUUID(id),
		UserID: database.ToPgxUUID(userID),
	}

	row, err := r.query.GetCategoryByID(ctx, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	result := &domain.Category{
		ID:           database.FromPgxUUID(row.ID),
		UserID:       database.FromPgxUUID(row.UserID),
		CategoryName: row.CategoryName,
	}

	return result, nil
}

func (r *categoryRepositoryImpl) ListCategories(ctx context.Context, userID uuid.UUID) ([]domain.Category, error) {
	rows, err := r.query.ListCategories(ctx, database.ToPgxUUID(userID))
	if err != nil {
		return nil, err
	}

	results := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		results = append(results, domain.Category{
			ID:           database.FromPgxUUID(row.ID),
			UserID:       database.FromPgxUUID(row.UserID),
			CategoryName: row.CategoryName,
		})
	}

	return results, nil
}

func (r *categoryRepositoryImpl) UpdateCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID, patch domain.CategoryForm) (*domain.Category, error) {
	args := sqlc.UpdateCategoryParams{
		CategoryName: patch.CategoryName,
		ID:           database.ToPgxUUID(id),
		UserID:       database.ToPgxUUID(userID),
	}

	row, err := r.query.UpdateCategory(ctx, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr)) && (pgErr.Code == "23505") {
			// Duplicate name
			return nil, domain.ErrConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	result := &domain.Category{
		ID:           database.FromPgxUUID(row.ID),
		UserID:       database.FromPgxUUID(row.UserID),
		CategoryName: row.CategoryName,
	}
	return result, nil
}

func (r *categoryRepositoryImpl) DeleteCategory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := sqlc.DeleteCategoryParams{
		ID:     database.ToPgxUUID(id),
		UserID: database.ToPgxUUID(userID),
	}

	rowCnt, err := r.query.DeleteCategory(ctx, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr)) && (pgErr.Code == "23503") {
			// Remove used category
			return domain.ErrConflict
		}
		return err
	}
	if rowCnt == 0 {
		// Not removing any category -> wrong ID / other user's category
		return domain.ErrNotFound
	}
	return nil
}
