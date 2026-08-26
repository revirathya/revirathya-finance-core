package postgres

import (
	"context"

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

// Sanity Check
var _ domain.CategoryRepository = (&categoryRepositoryImpl{})
