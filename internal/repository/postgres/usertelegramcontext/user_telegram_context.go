package usertelegramcontext

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	// Generated
	"github.com/revirathya/revirathya-finance-core/gen/sqlc"

	// Repository
	"github.com/revirathya/revirathya-finance-core/internal/domain"
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
			return nil, domain.ErrNotFound
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
			return nil, domain.ErrNotFound
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

func (r *userTelegramContextRepositoryImpl) UpdateUserTelegramContext(ctx context.Context, userID uuid.UUID, conversation *string, state *string, interactionChatID *string, interactionMessageID *string) (*domain.UserTelegramContext, error) {
	args := sqlc.UpdateUserTelegramContextParams{
		Conversation:         conversation,
		State:                state,
		InteractionChatID:    interactionChatID,
		InteractionMessageID: interactionMessageID,
		UserID:               database.ToPgxUUID(userID),
	}

	row, err := r.query.UpdateUserTelegramContext(ctx, args)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
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
