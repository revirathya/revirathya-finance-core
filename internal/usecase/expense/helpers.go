package expense

import "github.com/revirathya/revirathya-finance-core/internal/domain"

func sameExpense(form domain.ExpenseForm, expense domain.Expense) bool {
	if form.CategoryID != expense.CategoryID {
		return false
	}

	if !form.TransactionDt.Equal(expense.TransactionDt) {
		return false
	}

	if !form.Amount.Equal(expense.Amount) {
		return false
	}

	description := ""
	if form.Description != nil {
		description = *form.Description
	}
	if description != expense.Description {
		return false
	}

	return true
}
