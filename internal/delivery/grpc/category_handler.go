package grpc

import (
	"context"

	"github.com/google/uuid"
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/category"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	categories, err := h.ctgUc.ListCategories(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list categories: %v", err)
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
	uuidID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	category, err := h.ctgUc.GetCategoryByID(ctx, uuidID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get category by id: %v", err)
	}

	resp := &financecorev1.GetCategoryByIDResponse{
		Id:           category.ID.String(),
		CategoryName: category.CategoryName,
	}
	return resp, nil
}
