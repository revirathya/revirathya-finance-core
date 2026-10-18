package user

import (
	"context"
	"errors"
	"strings"

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

type userRepositoryImpl struct {
	query *sqlc.Queries
}

func NewUserRepositoryImpl(pool *pgxpool.Pool) domain.UserRepository {
	return &userRepositoryImpl{
		query: sqlc.New(pool),
	}
}

func (r *userRepositoryImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := r.query.GetUserByID(ctx, database.ToPgxUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
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

func (r *userRepositoryImpl) GetUserByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	row, err := r.query.GetUserByGoogleSub(ctx, &googleSub)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
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

func (r *userRepositoryImpl) CreateUser(ctx context.Context, form domain.UserForm) (*domain.User, error) {
	args := sqlc.CreateUserParams{
		FirstName: form.FirstName,
		LastName:  form.LastName,
		Email:     strings.ToLower(form.Email),
		GoogleSub: &form.GoogleSub,
	}

	var pgErr *pgconn.PgError
	row, err := r.query.CreateUser(ctx, args)
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // pgErr: Conflict
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	return r.GetUserByID(ctx, database.FromPgxUUID(row))
}

func (r *userRepositoryImpl) UpdateUser(ctx context.Context, id uuid.UUID, firstName string, lastName *string) (*domain.User, error) {
	args := sqlc.UpdateUserParams{
		FirstName: firstName,
		LastName:  lastName,
		ID:        database.ToPgxUUID(id),
	}

	row, err := r.query.UpdateUser(ctx, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return r.GetUserByID(ctx, database.FromPgxUUID(row))
}
