package telegrambindcode

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
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
)

type telegramBindCodeRepositoryImpl struct {
	pool  *pgxpool.Pool
	query *sqlc.Queries
}

func NewTelegramBindCodeRepositoryImpl(pool *pgxpool.Pool) domain.TelegramBindCodeRepository {
	return &telegramBindCodeRepositoryImpl{
		pool:  pool,
		query: sqlc.New(pool),
	}
}

func (r *telegramBindCodeRepositoryImpl) CreateBindCode(ctx context.Context, userID uuid.UUID, codeHash string, expiresAt time.Time) error {
	// Set transaction context to enable rollback
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Query inside transaction context
	q := r.query.WithTx(tx)

	// Remove user's bind code before create new
	if err := q.DeleteBindCodes(ctx, database.ToPgxUUID(userID)); err != nil {
		return err
	}

	// Create new bind code for user
	args := sqlc.CreateBindCodeParams{
		UserID:    database.ToPgxUUID(userID),
		CodeHash:  codeHash,
		ExpiresAt: database.ToPgxTimestamp(expiresAt),
	}
	if err := q.CreateBindCode(ctx, args); err != nil {
		return err
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *telegramBindCodeRepositoryImpl) BindCode(ctx context.Context, codeHash string, telegramID string) (*domain.User, error) {
	// Set transaction context to enable rollback
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Query inside transaction context
	q := r.query.WithTx(tx)

	// Consume bind code
	userID, err := q.ConsumeBindCode(ctx, codeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	// Update user's telegram binding
	args := sqlc.UpsertTelegramBindingParams{
		UserID:         userID,
		TelegramChatID: &telegramID,
	}
	userID, err = q.UpsertTelegramBinding(ctx, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr)) && (pgErr.Code == "23505") {
			// Telegram ID are used by other users
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Return user
	row, err := r.query.GetUserByID(ctx, userID)
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

func (r *telegramBindCodeRepositoryImpl) UnbindTelegram(ctx context.Context, userID uuid.UUID) error {
	if err := r.query.UnbindTelegramBinding(ctx, database.ToPgxUUID(userID)); err != nil {
		return err
	}
	return nil
}
