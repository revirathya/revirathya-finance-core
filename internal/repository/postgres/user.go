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

type userRepositoryImpl struct {
	query *sqlcgen.Queries
}

func NewUserRepositoryImpl(pool *pgxpool.Pool) domain.UserRepository {
	return &userRepositoryImpl{
		query: sqlcgen.New(pool),
	}
}

func (r *userRepositoryImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := r.query.GetUserByID(ctx, database.ToPgxUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
	}

	result := &domain.User{
		ID:             database.FromPgxUUID(row.ID),
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		Email:          row.Email,
		TelegramChatID: row.TelegramChatID,
	}
	return result, nil
}
