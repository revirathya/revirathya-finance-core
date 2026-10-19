package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

func ptr[T any](v T) *T { return &v }

// wantField fails unless err is a *ValidationError for field.
func wantField(t *testing.T, err error, field string) {
	t.Helper()
	var vErr *errDom.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("got %v, want a *ValidationError for %q", err, field)
	}
	if vErr.Field != field {
		t.Fatalf("field %q (%v), want %q", vErr.Field, err, field)
	}
}

var (
	validDt     = time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	validAmount = decimal.RequireFromString("42.50")
	tooLong100  = strings.Repeat("a", 101)
)

func TestCategoryValidate(t *testing.T) {
	t.Run("form trims the name in place", func(t *testing.T) {
		form := domain.CategoryForm{CategoryName: "  Food  "}
		if err := form.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if form.CategoryName != "Food" {
			t.Fatalf("category_name %q, want %q", form.CategoryName, "Food")
		}
	})

	t.Run("patch trims the name in place", func(t *testing.T) {
		patch := domain.CategoryPatch{CategoryName: "  Food  "}
		if err := patch.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patch.CategoryName != "Food" {
			t.Fatalf("category_name %q, want %q", patch.CategoryName, "Food")
		}
	})

	for _, name := range []string{"", "   ", tooLong100} {
		form := domain.CategoryForm{CategoryName: name}
		wantField(t, form.Validate(), "category_name")
		patch := domain.CategoryPatch{CategoryName: name}
		wantField(t, patch.Validate(), "category_name")
	}
}

func TestExpenseValidate(t *testing.T) {
	t.Run("valid form is accepted and the description trimmed", func(t *testing.T) {
		form := domain.ExpenseForm{UserID: uuid.New(), CategoryID: uuid.New(), TransactionDt: validDt, Amount: validAmount, Description: ptr("  Lunch  ")}
		if err := form.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if form.Description == nil || *form.Description != "Lunch" {
			t.Fatalf("description %v, want %q", form.Description, "Lunch")
		}
	})

	t.Run("nil description stays nil", func(t *testing.T) {
		form := domain.ExpenseForm{TransactionDt: validDt, Amount: validAmount}
		if err := form.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if form.Description != nil {
			t.Fatalf("description %q, want nil", *form.Description)
		}
	})

	// expenses.amount is NUMERIC(14, 2): anything below 10^12 must be accepted.
	t.Run("largest amount the column holds is accepted", func(t *testing.T) {
		form := domain.ExpenseForm{TransactionDt: validDt, Amount: decimal.RequireFromString("999999999999.99")}
		if err := form.Validate(); err != nil {
			t.Fatalf("form: unexpected error: %v", err)
		}
		patch := domain.ExpensePatch{TransactionDt: validDt, Amount: decimal.RequireFromString("999999999999.99")}
		if err := patch.Validate(); err != nil {
			t.Fatalf("patch: unexpected error: %v", err)
		}
	})

	invalid := []struct {
		name        string
		dt          time.Time
		amount      string
		description *string
		field       string
	}{
		{name: "zero amount", dt: validDt, amount: "0", field: "amount"},
		{name: "too many decimals", dt: validDt, amount: "1.005", field: "amount"},
		{name: "amount of 10^12", dt: validDt, amount: "1000000000000", field: "amount"},
		{name: "date before 2000", dt: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), amount: "1", field: "transaction_dt"},
		{name: "date far in the future", dt: time.Now().AddDate(0, 0, 3), amount: "1", field: "transaction_dt"},
		{name: "description over 500 characters", dt: validDt, amount: "1", description: ptr(strings.Repeat("a", 501)), field: "description"},
	}
	for _, tt := range invalid {
		t.Run("form rejects "+tt.name, func(t *testing.T) {
			form := domain.ExpenseForm{TransactionDt: tt.dt, Amount: decimal.RequireFromString(tt.amount), Description: tt.description}
			wantField(t, form.Validate(), tt.field)
		})
		t.Run("patch rejects "+tt.name, func(t *testing.T) {
			patch := domain.ExpensePatch{TransactionDt: tt.dt, Amount: decimal.RequireFromString(tt.amount), Description: tt.description}
			wantField(t, patch.Validate(), tt.field)
		})
	}

	t.Run("description of exactly 500 characters is accepted", func(t *testing.T) {
		patch := domain.ExpensePatch{TransactionDt: validDt, Amount: validAmount, Description: ptr(strings.Repeat("a", 500))}
		if err := patch.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestUserValidate(t *testing.T) {
	t.Run("form trims every field in place", func(t *testing.T) {
		form := domain.UserForm{FirstName: " Avi ", LastName: ptr(" Leonardi "), Email: " avi@example.com ", GoogleSub: " 123 "}
		if err := form.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if form.FirstName != "Avi" || *form.LastName != "Leonardi" || form.Email != "avi@example.com" || form.GoogleSub != "123" {
			t.Fatalf("got %q %q %q %q, want the values trimmed", form.FirstName, *form.LastName, form.Email, form.GoogleSub)
		}
	})

	t.Run("form rejects missing or too long fields", func(t *testing.T) {
		valid := func() domain.UserForm {
			return domain.UserForm{FirstName: "Avi", Email: "avi@example.com", GoogleSub: "123"}
		}
		cases := map[string]func(f *domain.UserForm){
			"first_name": func(f *domain.UserForm) { f.FirstName = "  " },
			"last_name":  func(f *domain.UserForm) { f.LastName = ptr(tooLong100) },
			"email":      func(f *domain.UserForm) { f.Email = "" },
			"google_sub": func(f *domain.UserForm) { f.GoogleSub = "" },
		}
		for field, mutate := range cases {
			form := valid()
			mutate(&form)
			wantField(t, form.Validate(), field)
		}
	})

	t.Run("patch trims names in place and keeps a nil last name nil", func(t *testing.T) {
		patch := domain.UserPatch{FirstName: " Avi "}
		if err := patch.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patch.FirstName != "Avi" || patch.LastName != nil {
			t.Fatalf("got %q %v, want %q and nil", patch.FirstName, patch.LastName, "Avi")
		}
	})

	t.Run("patch: a last name that is empty after trimming becomes nil (cleared)", func(t *testing.T) {
		for _, in := range []string{"", "   ", "\t\n"} {
			patch := domain.UserPatch{FirstName: "Avi", LastName: ptr(in)}
			if err := patch.Validate(); err != nil {
				t.Fatalf("%q: unexpected error: %v", in, err)
			}
			if patch.LastName != nil {
				t.Fatalf("%q: last name %q, want nil", in, *patch.LastName)
			}
		}
	})

	t.Run("patch: a non-empty last name is trimmed and kept", func(t *testing.T) {
		patch := domain.UserPatch{FirstName: "Avi", LastName: ptr("  Leonardi ")}
		if err := patch.Validate(); err != nil || patch.LastName == nil || *patch.LastName != "Leonardi" {
			t.Fatalf("got %v, %v; want Leonardi", patch.LastName, err)
		}
	})

	t.Run("patch rejects missing first name and too long names", func(t *testing.T) {
		empty := domain.UserPatch{FirstName: ""}
		wantField(t, empty.Validate(), "first_name")
		longFirst := domain.UserPatch{FirstName: tooLong100}
		wantField(t, longFirst.Validate(), "first_name")
		longLast := domain.UserPatch{FirstName: "Avi", LastName: ptr(tooLong100)}
		wantField(t, longLast.Validate(), "last_name")
	})
}

func TestUserTelegramContextValidate(t *testing.T) {
	t.Run("each field is trimmed into itself", func(t *testing.T) {
		patch := domain.UserTelegramContextPatch{
			Conversation:         ptr(" add_expense "),
			State:                ptr(" ask_amount "),
			InteractionChatID:    ptr(" 123 "),
			InteractionMessageID: ptr(" 456 "),
		}
		if err := patch.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *patch.Conversation != "add_expense" || *patch.State != "ask_amount" || *patch.InteractionChatID != "123" || *patch.InteractionMessageID != "456" {
			t.Fatalf("got %q %q %q %q, want each field trimmed in place",
				*patch.Conversation, *patch.State, *patch.InteractionChatID, *patch.InteractionMessageID)
		}
	})

	t.Run("nil fields stay nil", func(t *testing.T) {
		patch := domain.UserTelegramContextPatch{State: ptr("ask_amount")}
		if err := patch.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patch.Conversation != nil || patch.InteractionChatID != nil || patch.InteractionMessageID != nil {
			t.Fatalf("got %+v, want only state set", patch)
		}
	})

	cases := map[string]domain.UserTelegramContextPatch{
		"conversation":           {Conversation: ptr(tooLong100)},
		"state":                  {State: ptr(tooLong100)},
		"interaction_chat_id":    {InteractionChatID: ptr(tooLong100)},
		"interaction_message_id": {InteractionMessageID: ptr(tooLong100)},
	}
	for field, patch := range cases {
		wantField(t, patch.Validate(), field)
	}
}
