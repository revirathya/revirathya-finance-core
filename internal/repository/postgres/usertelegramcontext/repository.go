package usertelegramcontext

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	// Generated
	"github.com/revirathya/revirathya-finance-core/gen/sqlc"

	// Repository
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"
)

type userTelegramContextRepositoryImpl struct {
	query *sqlc.Queries
}

func NewUserTelegramContextRepositoryImpl(pool *pgxpool.Pool) domain.UserTelegramContextRepository {
	return &userTelegramContextRepositoryImpl{
		query: sqlc.New(pool),
	}
}

func (r *userTelegramContextRepositoryImpl) GetUserTelegramContext(ctx context.Context, userID uuid.UUID) (*domain.UserTelegramContext, error) {
	row, err := r.query.GetUserTelegramContext(ctx, database.ToPgxUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errDom.ErrNotFound
		}
		return nil, err
	}

	result := &domain.UserTelegramContext{
		ID:                   database.FromPgxUUID(row.ID),
		UserID:               database.FromPgxUUID(row.UserID),
		TelegramChatID:       row.TelegramChatID,
		Conversation:         row.Conversation,
		State:                row.State,
		InteractionChatID:    row.InteractionChatID,
		InteractionMessageID: row.InteractionMessageID,
	}
	return result, nil
}

func (r *userTelegramContextRepositoryImpl) GetUserTelegramContextByTelegramID(ctx context.Context, telegramChatID string) (*domain.UserTelegramContext, error) {
	row, err := r.query.GetUserTelegramContextByTelegramID(ctx, &telegramChatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errDom.ErrNotFound
		}
		return nil, err
	}

	result := &domain.UserTelegramContext{
		ID:                   database.FromPgxUUID(row.ID),
		UserID:               database.FromPgxUUID(row.UserID),
		TelegramChatID:       row.TelegramChatID,
		Conversation:         row.Conversation,
		State:                row.State,
		InteractionChatID:    row.InteractionChatID,
		InteractionMessageID: row.InteractionMessageID,
	}
	return result, nil
}

func (r *userTelegramContextRepositoryImpl) UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, patch domain.UserTelegramContextPatch) (*domain.UserTelegramContext, error) {
	args := sqlc.UpdateUserTelegramContextParams{
		Conversation:         patch.Conversation,
		State:                patch.State,
		InteractionChatID:    patch.InteractionChatID,
		InteractionMessageID: patch.InteractionMessageID,
		UserID:               database.ToPgxUUID(userID),
	}

	row, err := r.query.UpdateUserTelegramContext(ctx, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errDom.ErrNotFound
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "22001":
				// Too long or out of range
				return nil, errDom.ErrInvalidInput
			}
		}
		return nil, err
	}

	result := &domain.UserTelegramContext{
		ID:                   database.FromPgxUUID(row.ID),
		UserID:               database.FromPgxUUID(row.UserID),
		TelegramChatID:       row.TelegramChatID,
		Conversation:         row.Conversation,
		State:                row.State,
		InteractionChatID:    row.InteractionChatID,
		InteractionMessageID: row.InteractionMessageID,
	}
	return result, nil
}

// Sanity Check
var _ domain.UserTelegramContextRepository = (&userTelegramContextRepositoryImpl{})
