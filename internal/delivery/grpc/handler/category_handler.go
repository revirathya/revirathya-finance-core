package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"

	// Internal
	authApp "github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/category"

	// Internal - gRPC
	errGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
)

type CategoryHandler struct {
	financecorev1.UnimplementedCategoryServiceServer
	ctgUc *category.CategoryUsecase
}

func NewCategoryHandler(ctgUc *category.CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{
		ctgUc: ctgUc,
	}
}

func (h *CategoryHandler) ListCategories(ctx context.Context, req *financecorev1.ListCategoriesRequest) (*financecorev1.ListCategoriesResponse, error) {
	userID, err := authApp.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "list categories", err)
	}

	categories, err := h.ctgUc.ListCategories(ctx, userID)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "list categories", err)
	}

	resp := &financecorev1.ListCategoriesResponse{
		Categories: make([]*financecorev1.CategoryResponse, 0, len(categories)),
	}
	for _, c := range categories {
		resp.Categories = append(resp.Categories, &financecorev1.CategoryResponse{
			Id:           c.ID.String(),
			CategoryName: c.CategoryName,
		})
	}

	return resp, nil
}

func (h *CategoryHandler) GetCategoryByID(ctx context.Context, req *financecorev1.GetCategoryByIDRequest) (*financecorev1.GetCategoryByIDResponse, error) {
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	userID, err := authApp.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "get category by id", err)
	}

	category, err := h.ctgUc.GetCategoryByID(ctx, uuidId, userID)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("get category by id %s", uuidId.String()), err)
	}

	response := &financecorev1.GetCategoryByIDResponse{
		Id:           category.ID.String(),
		CategoryName: category.CategoryName,
	}
	return response, nil
}
