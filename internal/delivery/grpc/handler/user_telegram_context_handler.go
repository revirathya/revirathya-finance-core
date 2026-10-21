package handler

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"

	// Internal - gRPC
	errGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
)

type UserTelegramContextHandler struct {
	financecorev1.UnimplementedUserTelegramContextServiceServer
	telegramBindCode    telegramBindCodeUsecase
	userTelegramContext userTelegramContextUsecase
	bindLimiter         *ratelimit.Limiter
}

func NewUserTelegramContextHandler(telegramBindCode telegramBindCodeUsecase, userTelegramContext userTelegramContextUsecase, bindLimiter *ratelimit.Limiter) *UserTelegramContextHandler {
	return &UserTelegramContextHandler{
		telegramBindCode:    telegramBindCode,
		userTelegramContext: userTelegramContext,
		bindLimiter:         bindLimiter,
	}
}

func toProtoUserTelegramContext(utc *domain.UserTelegramContext) *financecorev1.UserTelegramContext {
	return &financecorev1.UserTelegramContext{
		Id:                   utc.ID.String(),
		UserId:               utc.UserID.String(),
		TelegramChatId:       utc.TelegramChatID,
		Conversation:         utc.Conversation,
		State:                utc.State,
		InteractionChatId:    utc.InteractionChatID,
		InteractionMessageId: utc.InteractionMessageID,
	}
}

func (h *UserTelegramContextHandler) GetUserTelegramContext(ctx context.Context, req *financecorev1.GetUserTelegramContextRequest) (*financecorev1.GetUserTelegramContextResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "get user telegram context", err)
	}

	// Parse input
	user_telegram_context, err := h.userTelegramContext.GetUserTelegramContext(ctx, userID)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("get user telegram context %s", userID.String()), err)
	}

	resp := &financecorev1.GetUserTelegramContextResponse{UserTelegramContext: toProtoUserTelegramContext(user_telegram_context)}
	return resp, nil
}

func (h *UserTelegramContextHandler) UpdateUserTelegramContext(ctx context.Context, req *financecorev1.UpdateUserTelegramContextRequest) (*financecorev1.UpdateUserTelegramContextResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update user telegram context", err)
	}

	// Parse input
	patch := domain.UserTelegramContextPatch{
		Conversation:         req.Conversation,
		State:                req.State,
		InteractionChatID:    req.InteractionChatId,
		InteractionMessageID: req.InteractionMessageId,
	}
	user_telegram_context, err := h.userTelegramContext.UpdateUserTelegramContext(ctx, userID, patch)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("update user telegram context %s", userID.String()), err)
	}

	resp := &financecorev1.UpdateUserTelegramContextResponse{UserTelegramContext: toProtoUserTelegramContext(user_telegram_context)}
	return resp, nil
}

func (h *UserTelegramContextHandler) BindTelegramAccount(ctx context.Context, req *financecorev1.BindTelegramAccountRequest) (*financecorev1.BindTelegramAccountResponse, error) {
	telegramID, err := auth.TelegramUserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "bind telegram account", err)
	}

	if !h.bindLimiter.Allow(telegramID) {
		return nil, status.Error(codes.ResourceExhausted, "too many bind attempts, try again later")
	}

	user, err := h.telegramBindCode.BindTelegram(ctx, req.Code, telegramID)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "bind telegram account", err)
	}

	resp := &financecorev1.BindTelegramAccountResponse{
		UserId:    user.ID.String(),
		FirstName: user.FirstName,
	}
	return resp, nil
}
