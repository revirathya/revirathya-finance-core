package handler

import (
	"context"
	"fmt"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/domain"

	// Internal - gRPC
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/converter"
	errGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
)

type CategoryHandler struct {
	financecorev1.UnimplementedCategoryServiceServer
	category categoryUsecase
}

func NewCategoryHandler(ctgUc categoryUsecase) *CategoryHandler {
	return &CategoryHandler{
		category: ctgUc,
	}
}

func toProtoCategory(c *domain.Category) *financecorev1.Category {
	return &financecorev1.Category{
		Id:           c.ID.String(),
		CategoryName: c.CategoryName,
	}
}

func (h *CategoryHandler) ListCategories(ctx context.Context, req *financecorev1.ListCategoriesRequest) (*financecorev1.ListCategoriesResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "list categories", err)
	}

	// List categories
	categoryPage, err := h.category.ListCategories(ctx, userID, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "list categories", err)
	}

	// Parse response
	response := &financecorev1.ListCategoriesResponse{
		Categories: make([]*financecorev1.Category, 0, len(categoryPage.Categories)),
	}
	for _, category := range categoryPage.Categories {
		response.Categories = append(response.Categories, toProtoCategory(&category))
	}

	if categoryPage.NextCursor != nil {
		response.NextPageToken = categoryPage.NextCursor.Encode()
	}

	return response, nil
}

func (h *CategoryHandler) GetCategoryByID(ctx context.Context, req *financecorev1.GetCategoryByIDRequest) (*financecorev1.GetCategoryByIDResponse, error) {
	id, err := converter.ParseUUID("id", req.GetId())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "get category by id", err)
	}

	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "get category by id", err)
	}

	category, err := h.category.GetCategoryByID(ctx, id, userID)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("get category by id %s", id.String()), err)
	}

	response := &financecorev1.GetCategoryByIDResponse{Category: toProtoCategory(category)}
	return response, nil
}
