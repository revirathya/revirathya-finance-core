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

	// Internal - gRPC
	"github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/converter"
	errGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc/error"
)

type ExpenseHandler struct {
	financecorev1.UnimplementedExpenseServiceServer
	expense expenseUsecase
}

func NewExpenseHandler(expense expenseUsecase) *ExpenseHandler {
	return &ExpenseHandler{
		expense: expense,
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
	categoryID, err := converter.ParseUUID("category_id", req.GetCategoryId())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "create expense", err)
	}

	transactionDt, err := converter.FromProtoDate("transaction_dt", req.GetTransactionDt())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "create expense", err)
	}

	amount, err := converter.FromProtoDecimal("amount", req.GetAmount())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "create expense", err)
	}

	// Create expense
	expense, err := h.expense.CreateExpense(ctx,
		userID,
		categoryID,
		transactionDt,
		req.Description,
		amount,
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
	id, err := converter.ParseUUID("id", req.GetId())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "get expense by id", err)
	}

	// Get expense by ID
	expense, err := h.expense.GetExpenseByID(ctx, id, userID)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("get expense %s", id.String()), err)
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
	expensePage, err := h.expense.ListExpenses(ctx, userID, tmTransactionDt, int(req.PageSize), req.PageToken)
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
	id, err := converter.ParseUUID("id", req.GetId())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update expense", err)
	}

	categoryID, err := converter.ParseOptionalUUID("category_id", req.CategoryId)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update expense", err)
	}

	transactionDt, err := converter.FromOptionalProtoDate("transaction_dt", req.GetTransactionDt())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update expense", err)
	}

	amount, err := converter.FromOptionalProtoDecimal("amount", req.GetAmount())
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, "update expense", err)
	}

	// Update expense
	patch := domain.ExpensePatch{
		CategoryID:    categoryID,
		TransactionDt: transactionDt,
		Description:   req.Description,
		Amount:        amount,
	}
	expense, err := h.expense.UpdateExpense(ctx, id, userID, patch)
	if err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("update expense %s", id.String()), err)
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
	id, err := converter.ParseUUID("id", req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id UUID format: %v", err)
	}

	// Delete expense
	if err := h.expense.DeleteExpense(ctx, id, userID); err != nil {
		return nil, errGrpc.ToStatus(ctx, fmt.Sprintf("delete expense %s", id.String()), err)
	}

	return &financecorev1.DeleteExpenseResponse{}, nil
}
