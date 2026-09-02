package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/expense"
	"github.com/shopspring/decimal"
)

type ExpenseHandler struct {
	financecorev1.UnimplementedExpenseServiceServer
	expUc *expense.ExpenseUsecase
}

func NewExpenseHandler(expUc *expense.ExpenseUsecase) *ExpenseHandler {
	return &ExpenseHandler{
		expUc: expUc,
	}
}

func (h *ExpenseHandler) CreateExpense(ctx context.Context, req *financecorev1.CreateExpenseRequest) (*financecorev1.CreateExpenseResponse, error) {
	uuidUserId, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id UUID format: %v", err)
	}

	uuidCategoryId, err := uuid.Parse(req.CategoryId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category_id UUID format: %v", err)
	}

	tmTransactionDt, err := time.Parse("2006-01-02", req.TransactionDt)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction_dt Time format (2006-01-02): %v", err)
	}

	decAmount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount Decimal format: %v", err)
	}

	expense, err := h.expUc.CreateExpense(ctx,
		uuidUserId,
		uuidCategoryId,
		tmTransactionDt,
		req.Description,
		decAmount,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create expense: %v", err)
	}

	resp := &financecorev1.CreateExpenseResponse{
		Id:            expense.ID.String(),
		UserId:        expense.UserID.String(),
		CategoryId:    expense.CategoryID.String(),
		TransactionDt: expense.TransactionDt.Format("2006-01-02"),
		Description:   expense.Description,
		Amount:        expense.Amount.String(),
	}
	return resp, nil
}

func (h *ExpenseHandler) GetExpenseByID(ctx context.Context, req *financecorev1.GetExpenseByIDRequest) (*financecorev1.GetExpenseByIDResponse, error) {
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	expense, err := h.expUc.GetExpenseByID(ctx, uuidId)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "expense %s not found", uuidId)
		}
		return nil, status.Errorf(codes.Internal, "failed to get expense by id: %v", err)
	}

	resp := &financecorev1.GetExpenseByIDResponse{
		Id:            expense.ID.String(),
		UserId:        expense.UserID.String(),
		CategoryId:    expense.CategoryID.String(),
		CategoryName:  expense.CategoryName,
		TransactionDt: expense.TransactionDt.Format("2006-01-02"),
		Description:   expense.Description,
		Amount:        expense.Amount.String(),
	}

	return resp, nil
}

func (h *ExpenseHandler) ListExpenses(ctx context.Context, req *financecorev1.ListExpensesRequest) (*financecorev1.ListExpensesResponse, error) {
	uuidUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id UUID format: %v", err)
	}

	var ntmTransactionDt *time.Time
	if req.TransactionDt != nil {
		tmTransactionDt, err := time.Parse("2006-01-02", *req.TransactionDt)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid transaction_dt Time format (2006-01-02): %v", err)
		}
		ntmTransactionDt = &tmTransactionDt
	}

	expenses, err := h.expUc.ListExpenses(ctx, uuidUserID, ntmTransactionDt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list expenses: %v", err)
	}

	resp := &financecorev1.ListExpensesResponse{
		Expenses: make([]*financecorev1.ExpenseResponse, 0, len(expenses)),
	}
	for _, e := range expenses {
		resp.Expenses = append(resp.Expenses, &financecorev1.ExpenseResponse{
			Id:            e.ID.String(),
			UserId:        e.UserID.String(),
			CategoryId:    e.CategoryID.String(),
			CategoryName:  e.CategoryName,
			TransactionDt: e.TransactionDt.Format("2006-01-02"),
			Description:   e.Description,
			Amount:        e.Amount.String(),
		})
	}

	return resp, nil
}

func (h *ExpenseHandler) UpdateExpense(ctx context.Context, req *financecorev1.UpdateExpenseRequest) (*financecorev1.UpdateExpenseResponse, error) {
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	uuidUserId, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id UUID format: %v", err)
	}

	uuidCategoryId, err := uuid.Parse(req.CategoryId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category_id UUID format: %v", err)
	}

	tmTransactionDt, err := time.Parse("2006-01-02", req.TransactionDt)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction_dt Time format (2006-01-02): %v", err)
	}

	decAmount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount Decimal format: %v", err)
	}

	expense, err := h.expUc.UpdateExpense(
		ctx,
		uuidId,
		uuidUserId,
		uuidCategoryId,
		tmTransactionDt,
		req.Description,
		decAmount,
	)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "expense %s not found", uuidId)
		}
		return nil, status.Errorf(codes.Internal, "failed to update expense: %v", err)
	}

	resp := &financecorev1.UpdateExpenseResponse{
		Id:            expense.ID.String(),
		UserId:        expense.UserID.String(),
		CategoryId:    expense.CategoryID.String(),
		TransactionDt: expense.TransactionDt.Format("2006-01-02"),
		Description:   expense.Description,
		Amount:        expense.Amount.String(),
	}

	return resp, nil
}

func (h *ExpenseHandler) DeleteExpense(ctx context.Context, req *financecorev1.DeleteExpenseRequest) (*financecorev1.DeleteExpenseResponse, error) {
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	if err := h.expUc.DeleteExpense(ctx, uuidId); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "expense %s not found", uuidId)
		}
		return nil, status.Errorf(codes.Internal, "failed to delete expense: %v", err)
	}

	return &financecorev1.DeleteExpenseResponse{}, nil
}
