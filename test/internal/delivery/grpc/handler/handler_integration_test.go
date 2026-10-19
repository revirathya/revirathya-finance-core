//go:build integration

package handler_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/go/financecore/v1"

	// Internal
	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/usecase"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

const apiKey = "test-api-key"

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}

// bot is a client of the production gRPC server (dlvGrpc.Setup) on the real usecases and Postgres, acting as the
// Telegram bot: every call carries the API key and the sender's Telegram ID.
type bot struct {
	t          *testing.T
	uc         usecase.Usecase
	categories financecorev1.CategoryServiceClient
	expenses   financecorev1.ExpenseServiceClient
	contexts   financecorev1.UserTelegramContextServiceClient
}

func newBot(t *testing.T) *bot {
	t.Helper()
	uc := testutil.NewUsecases(testutil.NewFakeGoogleVerifier())
	layer, err := dlvGrpc.Setup("127.0.0.1:0", false, apiKey, metrics.NewMetrics(nil), uc)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	go func() { _ = layer.Server.Serve(layer.Listener) }()
	t.Cleanup(layer.Server.Stop)

	conn, err := grpc.NewClient(layer.Listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &bot{
		t: t, uc: uc,
		categories: financecorev1.NewCategoryServiceClient(conn),
		expenses:   financecorev1.NewExpenseServiceClient(conn),
		contexts:   financecorev1.NewUserTelegramContextServiceClient(conn),
	}
}

// as returns a context for a call sent by the given Telegram user.
func (b *bot) as(telegramID string) context.Context {
	b.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	b.t.Cleanup(cancel)
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+apiKey, "x-telegram-user-id", telegramID)
}

// boundUser creates a user and binds a new Telegram account to them through BindTelegramAccount, the way the bot
// does after the user sends /bind <code>. It returns the user and their Telegram ID.
func (b *bot) boundUser() (*domain.User, string) {
	b.t.Helper()
	user := testutil.NewUser(b.t)
	code, _, err := b.uc.TelegramBindCode.CreateBindCode(context.Background(), user.ID)
	if err != nil {
		b.t.Fatalf("create bind code: %v", err)
	}
	telegramID := testutil.NewTelegramID()
	if _, err := b.contexts.BindTelegramAccount(b.as(telegramID), &financecorev1.BindTelegramAccountRequest{Code: code}); err != nil {
		b.t.Fatalf("bind: %v", err)
	}
	return user, telegramID
}

func wantCode(t *testing.T, err error, code codes.Code, message string) {
	t.Helper()
	st, _ := status.FromError(err)
	if st.Code() != code {
		t.Fatalf("code %v (%v), want %v", st.Code(), err, code)
	}
	if message != "" && !strings.Contains(st.Message(), message) {
		t.Fatalf("message %q, want it to contain %q", st.Message(), message)
	}
}

func TestBindTelegramAccount(t *testing.T) {
	b := newBot(t)

	t.Run("a valid code binds the sender and returns the user", func(t *testing.T) {
		user := testutil.NewUser(t)
		code, _, _ := b.uc.TelegramBindCode.CreateBindCode(context.Background(), user.ID)
		telegramID := testutil.NewTelegramID()

		resp, err := b.contexts.BindTelegramAccount(b.as(telegramID), &financecorev1.BindTelegramAccountRequest{Code: code})

		if err != nil {
			t.Fatalf("bind: %v", err)
		}
		if resp.GetUserId() != user.ID.String() || resp.GetFirstName() != user.FirstName {
			t.Fatalf("got %v, want user %v", resp, user.ID)
		}
		ctx, err := b.contexts.GetUserTelegramContext(b.as(telegramID), &financecorev1.GetUserTelegramContextRequest{})
		if err != nil || ctx.GetTelegramChatId() != telegramID {
			t.Fatalf("context after bind: %v, %v; want chat %s", ctx, err, telegramID)
		}
	})

	t.Run("a code works once", func(t *testing.T) {
		user := testutil.NewUser(t)
		code, _, _ := b.uc.TelegramBindCode.CreateBindCode(context.Background(), user.ID)
		if _, err := b.contexts.BindTelegramAccount(b.as(testutil.NewTelegramID()), &financecorev1.BindTelegramAccountRequest{Code: code}); err != nil {
			t.Fatalf("first bind: %v", err)
		}

		_, err := b.contexts.BindTelegramAccount(b.as(testutil.NewTelegramID()), &financecorev1.BindTelegramAccountRequest{Code: code})
		wantCode(t, err, codes.InvalidArgument, "")
	})

	t.Run("a malformed or unknown code is INVALID_ARGUMENT", func(t *testing.T) {
		for _, code := range []string{"garbage", "abcd-efgh", "ZZZZ-ZZZZ"} {
			_, err := b.contexts.BindTelegramAccount(b.as(testutil.NewTelegramID()), &financecorev1.BindTelegramAccountRequest{Code: code})
			wantCode(t, err, codes.InvalidArgument, "")
		}
	})

	t.Run("a Telegram account already bound to another user is ALREADY_EXISTS", func(t *testing.T) {
		_, telegramID := b.boundUser()
		other := testutil.NewUser(t)
		code, _, _ := b.uc.TelegramBindCode.CreateBindCode(context.Background(), other.ID)

		_, err := b.contexts.BindTelegramAccount(b.as(telegramID), &financecorev1.BindTelegramAccountRequest{Code: code})
		wantCode(t, err, codes.AlreadyExists, "")
	})

	t.Run("more than 10 attempts in 5 minutes is RESOURCE_EXHAUSTED", func(t *testing.T) {
		telegramID := testutil.NewTelegramID()
		var err error
		for range 11 {
			_, err = b.contexts.BindTelegramAccount(b.as(telegramID), &financecorev1.BindTelegramAccountRequest{Code: "ZZZZ-ZZZZ"})
		}
		wantCode(t, err, codes.ResourceExhausted, "")
	})
}

func TestAuthentication(t *testing.T) {
	b := newBot(t)

	t.Run("an unbound Telegram account may only bind", func(t *testing.T) {
		_, err := b.categories.ListCategories(b.as(testutil.NewTelegramID()), &financecorev1.ListCategoriesRequest{})
		wantCode(t, err, codes.Unauthenticated, "")
	})

	t.Run("a wrong API key is rejected before anything else", func(t *testing.T) {
		_, telegramID := b.boundUser()
		ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer wrong", "x-telegram-user-id", telegramID)

		_, err := b.categories.ListCategories(ctx, &financecorev1.ListCategoriesRequest{})
		wantCode(t, err, codes.Unauthenticated, "")
	})
}

func TestCategoryHandler(t *testing.T) {
	b := newBot(t)
	alice, aliceTG := b.boundUser()
	bob, _ := b.boundUser()
	food := testutil.NewCategory(t, alice.ID, "Food")
	testutil.NewCategory(t, alice.ID, "Bills")
	bobsFood := testutil.NewCategory(t, bob.ID, "Food")

	t.Run("list returns only the caller's categories, by name", func(t *testing.T) {
		resp, err := b.categories.ListCategories(b.as(aliceTG), &financecorev1.ListCategoriesRequest{})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		got := resp.GetCategories()
		if len(got) != 2 || got[0].GetCategoryName() != "Bills" || got[1].GetCategoryName() != "Food" {
			t.Fatalf("categories %v, want Bills then Food", got)
		}
	})

	t.Run("get returns the caller's category; another user's is NOT_FOUND", func(t *testing.T) {
		resp, err := b.categories.GetCategoryByID(b.as(aliceTG), &financecorev1.GetCategoryByIDRequest{Id: food.ID.String()})
		if err != nil || resp.GetId() != food.ID.String() || resp.GetCategoryName() != "Food" {
			t.Fatalf("got %v, %v; want Food", resp, err)
		}

		_, err = b.categories.GetCategoryByID(b.as(aliceTG), &financecorev1.GetCategoryByIDRequest{Id: bobsFood.ID.String()})
		wantCode(t, err, codes.NotFound, "")
	})

	t.Run("a malformed ID is INVALID_ARGUMENT", func(t *testing.T) {
		_, err := b.categories.GetCategoryByID(b.as(aliceTG), &financecorev1.GetCategoryByIDRequest{Id: "nope"})
		wantCode(t, err, codes.InvalidArgument, "")
	})
}

func TestExpenseHandler(t *testing.T) {
	b := newBot(t)
	alice, aliceTG := b.boundUser()
	bob, bobTG := b.boundUser()
	food := testutil.NewCategory(t, alice.ID, "Food").ID.String()
	bobsFood := testutil.NewCategory(t, bob.ID, "Food").ID.String()
	str := func(s string) *string { return &s }

	t.Run("create returns the stored expense, with the caller as owner", func(t *testing.T) {
		resp, err := b.expenses.CreateExpense(b.as(aliceTG), &financecorev1.CreateExpenseRequest{
			CategoryId: food, TransactionDt: "2026-01-15", Description: str("  Lunch "), Amount: "12500.50",
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if resp.GetUserId() != alice.ID.String() || resp.GetCategoryId() != food || resp.GetTransactionDt() != "2026-01-15" ||
			resp.GetDescription() != "Lunch" || resp.GetAmount() != "12500.5" {
			t.Fatalf("got %v", resp)
		}

		got, err := b.expenses.GetExpenseByID(b.as(aliceTG), &financecorev1.GetExpenseByIDRequest{Id: resp.GetId()})
		if err != nil || got.GetCategoryName() != "Food" || got.GetDescription() != "Lunch" {
			t.Fatalf("get: %v, %v; want Food / Lunch", got, err)
		}
	})

	t.Run("create rejects bad input with INVALID_ARGUMENT and the field", func(t *testing.T) {
		for _, tt := range []struct {
			req     *financecorev1.CreateExpenseRequest
			message string
		}{
			{&financecorev1.CreateExpenseRequest{CategoryId: "nope", TransactionDt: "2026-01-15", Amount: "1"}, "category_id"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: "15-01-2026", Amount: "1"}, "transaction_dt"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: "2026-01-15", Amount: "ten"}, "amount"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: "2026-01-15", Amount: "0"}, "amount: must be greater than 0"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: "2026-01-15", Amount: "1.005"}, "amount: at most 2 decimal places"},
			{&financecorev1.CreateExpenseRequest{CategoryId: bobsFood, TransactionDt: "2026-01-15", Amount: "1"}, ""},
		} {
			_, err := b.expenses.CreateExpense(b.as(aliceTG), tt.req)
			wantCode(t, err, codes.InvalidArgument, tt.message)
		}
	})

	t.Run("update keeps an omitted description and clears it with an empty string", func(t *testing.T) {
		created, _ := b.expenses.CreateExpense(b.as(aliceTG), &financecorev1.CreateExpenseRequest{
			CategoryId: food, TransactionDt: "2026-01-15", Description: str("Lunch"), Amount: "10",
		})
		req := &financecorev1.UpdateExpenseRequest{Id: created.GetId(), CategoryId: food, TransactionDt: "2026-01-16", Amount: "20"}

		kept, err := b.expenses.UpdateExpense(b.as(aliceTG), req)
		if err != nil || kept.GetDescription() != "Lunch" || kept.GetAmount() != "20" || kept.GetTransactionDt() != "2026-01-16" {
			t.Fatalf("update without description: %v, %v", kept, err)
		}
		req.Description = str("")
		cleared, err := b.expenses.UpdateExpense(b.as(aliceTG), req)
		if err != nil || cleared.GetDescription() != "" {
			t.Fatalf("update with empty description: %v, %v", cleared, err)
		}
	})

	t.Run("another user's expense is NOT_FOUND; deleting twice is NOT_FOUND", func(t *testing.T) {
		created, _ := b.expenses.CreateExpense(b.as(aliceTG), &financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: "2026-01-15", Amount: "10"})
		id := created.GetId()

		_, err := b.expenses.GetExpenseByID(b.as(bobTG), &financecorev1.GetExpenseByIDRequest{Id: id})
		wantCode(t, err, codes.NotFound, "")
		_, err = b.expenses.DeleteExpense(b.as(bobTG), &financecorev1.DeleteExpenseRequest{Id: id})
		wantCode(t, err, codes.NotFound, "")

		if _, err := b.expenses.DeleteExpense(b.as(aliceTG), &financecorev1.DeleteExpenseRequest{Id: id}); err != nil {
			t.Fatalf("delete: %v", err)
		}
		_, err = b.expenses.DeleteExpense(b.as(aliceTG), &financecorev1.DeleteExpenseRequest{Id: id})
		wantCode(t, err, codes.NotFound, "")
	})

	t.Run("list pages with page_size / page_token, newest first, and filters by date", func(t *testing.T) {
		carol, carolTG := b.boundUser()
		carolsFood := testutil.NewCategory(t, carol.ID, "Food").ID.String()
		for _, dt := range []string{"2026-02-01", "2026-02-02", "2026-02-02", "2026-02-03", "2026-02-04"} {
			if _, err := b.expenses.CreateExpense(b.as(carolTG), &financecorev1.CreateExpenseRequest{CategoryId: carolsFood, TransactionDt: dt, Amount: "1"}); err != nil {
				t.Fatalf("create: %v", err)
			}
		}

		var dates []string
		seen := map[string]bool{}
		token := ""
		for page := 0; ; page++ {
			if page > 5 {
				t.Fatal("pagination did not end")
			}
			resp, err := b.expenses.ListExpenses(b.as(carolTG), &financecorev1.ListExpensesRequest{PageSize: 2, PageToken: token})
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			for _, e := range resp.GetExpenses() {
				if seen[e.GetId()] || e.GetUserId() != carol.ID.String() {
					t.Fatalf("expense %v repeated or not carol's", e.GetId())
				}
				seen[e.GetId()] = true
				dates = append(dates, e.GetTransactionDt())
			}
			if token = resp.GetNextPageToken(); token == "" {
				break
			}
		}
		if want := "2026-02-04 2026-02-03 2026-02-02 2026-02-02 2026-02-01"; strings.Join(dates, " ") != want {
			t.Fatalf("dates %v, want %s", dates, want)
		}

		filtered, err := b.expenses.ListExpenses(b.as(carolTG), &financecorev1.ListExpensesRequest{TransactionDt: str("2026-02-02")})
		if err != nil || len(filtered.GetExpenses()) != 2 || filtered.GetNextPageToken() != "" {
			t.Fatalf("filtered: %v, %v; want the 2 expenses on 2026-02-02 and no next page", filtered, err)
		}

		_, err = b.expenses.ListExpenses(b.as(carolTG), &financecorev1.ListExpensesRequest{PageToken: "garbage!"})
		wantCode(t, err, codes.InvalidArgument, "page_token")
		_, err = b.expenses.ListExpenses(b.as(carolTG), &financecorev1.ListExpensesRequest{PageSize: -1})
		wantCode(t, err, codes.InvalidArgument, "page_size")
	})
}

func TestUserTelegramContextHandler(t *testing.T) {
	b := newBot(t)
	_, tg := b.boundUser()
	str := func(s string) *string { return &s }

	t.Run("update sets only the fields sent, and get returns them", func(t *testing.T) {
		if _, err := b.contexts.UpdateUserTelegramContext(b.as(tg), &financecorev1.UpdateUserTelegramContextRequest{
			Conversation: str("add_expense"), State: str(" ask_amount "), InteractionChatId: str("1"), InteractionMessageId: str("2"),
		}); err != nil {
			t.Fatalf("first update: %v", err)
		}
		resp, err := b.contexts.UpdateUserTelegramContext(b.as(tg), &financecorev1.UpdateUserTelegramContextRequest{State: str("ask_category")})
		if err != nil {
			t.Fatalf("second update: %v", err)
		}

		got, err := b.contexts.GetUserTelegramContext(b.as(tg), &financecorev1.GetUserTelegramContextRequest{})
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		for _, c := range []*financecorev1.GetUserTelegramContextResponse{got, {
			Conversation: resp.Conversation, State: resp.State, InteractionChatId: resp.InteractionChatId, InteractionMessageId: resp.InteractionMessageId,
		}} {
			if c.GetConversation() != "add_expense" || c.GetState() != "ask_category" || c.GetInteractionChatId() != "1" || c.GetInteractionMessageId() != "2" {
				t.Fatalf("context %v, want conversation kept, state updated, interaction ids kept", c)
			}
		}
	})

	t.Run("a too long value is INVALID_ARGUMENT with the field", func(t *testing.T) {
		_, err := b.contexts.UpdateUserTelegramContext(b.as(tg), &financecorev1.UpdateUserTelegramContextRequest{State: str(strings.Repeat("a", 101))})
		wantCode(t, err, codes.InvalidArgument, "state: must be at most 100 characters")
	})

	t.Run("the context is the caller's: another bound user sees their own", func(t *testing.T) {
		_, otherTG := b.boundUser()
		got, err := b.contexts.GetUserTelegramContext(b.as(otherTG), &financecorev1.GetUserTelegramContextRequest{})
		if err != nil || got.GetTelegramChatId() != otherTG || got.GetState() != "" {
			t.Fatalf("got %v, %v; want the other user's fresh context", got, err)
		}
	})
}
