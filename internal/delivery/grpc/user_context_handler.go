package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	grpcErr "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/usercontext"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserTelegramContextHandler struct {
	financecorev1.UnimplementedUserTelegramContextServiceServer
	utelUc *usercontext.UserTelegramContextUsecase
}

func NewUserTelegramContextHandler(utelUc *usercontext.UserTelegramContextUsecase) *UserTelegramContextHandler {
	return &UserTelegramContextHandler{
		utelUc: utelUc,
	}
}

func (h *UserTelegramContextHandler) GetUserTelegramContext(ctx context.Context, req *financecorev1.GetUserTelegramContextRequest) (*financecorev1.GetUserTelegramContextResponse, error) {
	uuidUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id UUID format: %v", err)
	}

	user_context, err := h.utelUc.GetUserTelegramContext(ctx, uuidUserID)
	if err != nil {
		return nil, grpcErr.ToStatus(fmt.Sprintf("get user telegram context %s", uuidUserID.String()), err)
	}

	resp := &financecorev1.GetUserTelegramContextResponse{
		Id:                   user_context.ID.String(),
		UserId:               user_context.UserID.String(),
		TelegramChatId:       user_context.TelegramChatID,
		Conversation:         user_context.Conversation,
		State:                user_context.State,
		InteractionChatId:    user_context.InteractionChatID,
		InteractionMessageId: user_context.InteractionMessageID,
	}
	return resp, nil
}

func (h *UserTelegramContextHandler) GetUserTelegramContextByTelegramID(ctx context.Context, req *financecorev1.GetUserTelegramContextByTelegramIDRequest) (*financecorev1.GetUserTelegramContextByTelegramIDResponse, error) {
	user_context, err := h.utelUc.GetUserTelegramContextByTelegramID(ctx, req.TelegramChatId)
	if err != nil {
		return nil, grpcErr.ToStatus(fmt.Sprintf("get user telegram context by telegram id %s", req.TelegramChatId), err)
	}

	resp := &financecorev1.GetUserTelegramContextByTelegramIDResponse{
		Id:                   user_context.ID.String(),
		UserId:               user_context.UserID.String(),
		TelegramChatId:       user_context.TelegramChatID,
		Conversation:         user_context.Conversation,
		State:                user_context.State,
		InteractionChatId:    user_context.InteractionChatID,
		InteractionMessageId: user_context.InteractionMessageID,
	}
	return resp, nil
}

func (h *UserTelegramContextHandler) UpdateUserTelegramContext(ctx context.Context, req *financecorev1.UpdateUserTelegramContextRequest) (*financecorev1.UpdateUserTelegramContextResponse, error) {
	uuidUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id UUID format: %v", err)
	}

	user_context, err := h.utelUc.UpdateUserTelegramContext(
		ctx,
		uuidUserID,
		req.Conversation,
		req.State,
		req.InteractionChatId,
		req.InteractionMessageId,
	)
	if err != nil {
		return nil, grpcErr.ToStatus(fmt.Sprintf("update user telegram context %s", uuidUserID.String()), err)
	}

	resp := &financecorev1.UpdateUserTelegramContextResponse{
		Id:                   user_context.ID.String(),
		UserId:               user_context.UserID.String(),
		TelegramChatId:       user_context.TelegramChatID,
		Conversation:         user_context.Conversation,
		State:                user_context.State,
		InteractionChatId:    user_context.InteractionChatID,
		InteractionMessageId: user_context.InteractionMessageID,
	}

	return resp, nil
}
