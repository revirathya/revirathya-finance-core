package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/expense"

	// Internal - gRPC
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/converter"
	errGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
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

func toProtoExpense(e *domain.Expense) *financecorev1.Expense {
	return &financecorev1.Expense{
		Id:            e.ID.String(),
		UserId:        e.UserID.String(),
		CategoryId:    e.CategoryID.String(),
		CategoryName:  e.CategoryName,
		TransactionDt: converter.ToProtoDate(e.TransactionDt),
		Description:   e.Description,
		Amount:        converter.ToProtoDecimal(e.Amount),
	}
}

func (h *ExpenseHandler) CreateExpense(ctx context.Context, req *financecorev1.CreateExpenseRequest) (*financecorev1.CreateExpenseResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "create expense", err)
	}

	// Parse input
	uuidCategoryId, err := uuid.Parse(req.CategoryId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid category_id UUID format: %v", err)
	}

	tmTransactionDt, err := converter.FromProtoDate("transaction_dt", req.GetTransactionDt())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "create expense", err)
	}

	decAmount, err := converter.FromProtoDecimal("amount", req.GetAmount())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "create expense", err)
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
		return nil, errGrpc.ToStatus(ctx, "create expense", err)
	}

	resp := &financecorev1.CreateExpenseResponse{Expense: toProtoExpense(expense)}
	return resp, nil
}

func (h *ExpenseHandler) GetExpenseByID(ctx context.Context, req *financecorev1.GetExpenseByIDRequest) (*financecorev1.GetExpenseByIDResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "get expense by id", err)
	}

	// Parse input
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	// Get expense by ID
	expense, err := h.expUc.GetExpenseByID(ctx, uuidId, userID)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("get expense %s", uuidId.String()), err)
	}

	resp := &financecorev1.GetExpenseByIDResponse{Expense: toProtoExpense(expense)}
	return resp, nil
}

func (h *ExpenseHandler) ListExpenses(ctx context.Context, req *financecorev1.ListExpensesRequest) (*financecorev1.ListExpensesResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "list expenses", err)
	}

	// Parse input
	tmTransactionDt, err := converter.FromOptionalProtoDate("transaction_dt", req.GetTransactionDt())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "list expenses", err)
	}

	// List expenses
	expensePage, err := h.expUc.ListExpenses(ctx, userID, tmTransactionDt, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "list expenses", err)
	}

	resp := &financecorev1.ListExpensesResponse{
		Expenses: make([]*financecorev1.Expense, 0, len(expensePage.Expenses)),
	}
	for _, expense := range expensePage.Expenses {
		resp.Expenses = append(resp.Expenses, toProtoExpense(&expense))
	}

	if expensePage.NextCursor != nil {
		resp.NextPageToken = expensePage.NextCursor.Encode()
	}

	return resp, nil
}

func (h *ExpenseHandler) UpdateExpense(ctx context.Context, req *financecorev1.UpdateExpenseRequest) (*financecorev1.UpdateExpenseResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update expense", err)
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

	tmTransactionDt, err := converter.FromProtoDate("transaction_dt", req.GetTransactionDt())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update expense", err)
	}

	decAmount, err := converter.FromProtoDecimal("amount", req.GetAmount())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update expense", err)
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
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("update expense %s", uuidId.String()), err)
	}

	resp := &financecorev1.UpdateExpenseResponse{Expense: toProtoExpense(expense)}
	return resp, nil
}

func (h *ExpenseHandler) DeleteExpense(ctx context.Context, req *financecorev1.DeleteExpenseRequest) (*financecorev1.DeleteExpenseResponse, error) {
	// Get user from context
	userID, err := auth.UserIDFrom(ctx)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "delete expense", err)
	}

	// Parse input
	uuidId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	// Delete expense
	if err := h.expUc.DeleteExpense(ctx, uuidId, userID); err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("delete expense %s", uuidId.String()), err)
	}

	return &financecorev1.DeleteExpenseResponse{}, nil
}
