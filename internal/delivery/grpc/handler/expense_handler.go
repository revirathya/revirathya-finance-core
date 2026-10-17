package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"
	appAuth "github.com/revirathya/revirathya-finance-core/internal/auth"
	grpcErr "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
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
	// Get user from context
	userID, err := appAuth.UserIDFrom(ctx)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, "create expense", err)
	}

	// Parse input
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

	// Create expense
	expense, err := h.expUc.CreateExpense(ctx,
		userID,
		uuidCategoryId,
		tmTransactionDt,
		req.Description,
		decAmount,
	)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, "create expense", err)
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
	// Get user from context
	userID, err := appAuth.UserIDFrom(ctx)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, "get expense by id", err)
	}

	// Parse input
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	// Get expense by ID
	expense, err := h.expUc.GetExpenseByID(ctx, uuidId, userID)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, fmt.Sprintf("get expense %s", uuidId.String()), err)
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
	// Get user from context
	userID, err := appAuth.UserIDFrom(ctx)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, "list expenses", err)
	}

	// Parse input
	var ntmTransactionDt *time.Time
	if req.TransactionDt != nil {
		tmTransactionDt, err := time.Parse("2006-01-02", *req.TransactionDt)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid transaction_dt Time format (2006-01-02): %v", err)
		}
		ntmTransactionDt = &tmTransactionDt
	}

	// List expenses
	expenses, err := h.expUc.ListExpenses(ctx, userID, ntmTransactionDt)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, "list expenses", err)
	}

	resp := &financecorev1.ListExpensesResponse{
		Expenses: make([]*financecorev1.ExpenseResponse, 0, len(expenses)),
	}
	for _, expense := range expenses {
		resp.Expenses = append(resp.Expenses, &financecorev1.ExpenseResponse{
			Id:            expense.ID.String(),
			UserId:        expense.UserID.String(),
			CategoryId:    expense.CategoryID.String(),
			CategoryName:  expense.CategoryName,
			TransactionDt: expense.TransactionDt.Format("2006-01-02"),
			Description:   expense.Description,
			Amount:        expense.Amount.String(),
		})
	}

	return resp, nil
}

func (h *ExpenseHandler) UpdateExpense(ctx context.Context, req *financecorev1.UpdateExpenseRequest) (*financecorev1.UpdateExpenseResponse, error) {
	// Get user from context
	userID, err := appAuth.UserIDFrom(ctx)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, "update expense", err)
	}

	// Parse input
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
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

	// Update expense
	expense, err := h.expUc.UpdateExpense(
		ctx,
		uuidId,
		userID,
		uuidCategoryId,
		tmTransactionDt,
		req.Description,
		decAmount,
	)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, fmt.Sprintf("update expense %s", uuidId.String()), err)
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
	// Get user from context
	userID, err := appAuth.UserIDFrom(ctx)
	if err != nil {
		return nil, grpcErr.ToStatus(ctx, "delete expense", err)
	}

	// Parse input
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	// Delete expense
	if err := h.expUc.DeleteExpense(ctx, uuidId, userID); err != nil {
		return nil, grpcErr.ToStatus(ctx, fmt.Sprintf("delete expense %s", uuidId.String()), err)
	}

	return &financecorev1.DeleteExpenseResponse{}, nil
}
