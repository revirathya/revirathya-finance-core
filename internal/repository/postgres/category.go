package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
	sqlcgen "github.com/revirathya/revirathya-finance-core/internal/repository/sqlc/gen"
)

type categoryRepositoryImpl struct {
	query *sqlcgen.Queries
}

func NewCategoryRepositoryImpl(pool *pgxpool.Pool) domain.CategoryRepository {
	return &categoryRepositoryImpl{
		query: sqlcgen.New(pool),
	}
}

func (r *categoryRepositoryImpl) CreateCategory(ctx context.Context, input domain.CategoryForm) (*domain.Category, error) {
	row, err := r.query.CreateCategory(ctx, input.CategoryName)
	if err != nil {
		return nil, err
	}

	result := &domain.Category{
		ID:           database.FromPgxUUID(row.ID),
		CategoryName: row.CategoryName,
	}
	return result, nil
}

func (r *categoryRepositoryImpl) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	row, err := r.query.GetCategoryByID(ctx, database.ToPgxUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	result := &domain.Category{
		ID:           database.FromPgxUUID(row.ID),
		CategoryName: row.CategoryName,
	}

	return result, nil
}

func (r *categoryRepositoryImpl) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.query.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		results = append(results, domain.Category{
			ID:           database.FromPgxUUID(row.ID),
			CategoryName: row.CategoryName,
		})
	}

	return results, nil
}

func (r *categoryRepositoryImpl) UpdateCategory(ctx context.Context, id uuid.UUID, patch domain.CategoryForm) (*domain.Category, error) {
	args := sqlcgen.UpdateCategoryParams{
		ID:           database.ToPgxUUID(id),
		CategoryName: patch.CategoryName,
	}

	row, err := r.query.UpdateCategory(ctx, args)
	if err != nil {
		return nil, err
	}

	result := &domain.Category{
		ID:           database.FromPgxUUID(row.ID),
		CategoryName: row.CategoryName,
	}
	return result, nil
}

func (r *categoryRepositoryImpl) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if err := r.query.DeleteCategory(ctx, database.ToPgxUUID(id)); err != nil {
		return err
	}
	return nil
}
