//go:build integration

package handler_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/type/date"
	gdecimal "google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

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

// day builds a google.type.Date from "YYYY-MM-DD".
func day(t *testing.T, s string) *date.Date {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return &date.Date{Year: int32(d.Year()), Month: int32(d.Month()), Day: int32(d.Day())} //nolint:gosec // G115: test dates
}

// amount builds a google.type.Decimal.
func amount(v string) *gdecimal.Decimal { return &gdecimal.Decimal{Value: v} }

// ymd formats a google.type.Date as "YYYY-MM-DD".
func ymd(d *date.Date) string {
	return time.Date(int(d.GetYear()), time.Month(d.GetMonth()), int(d.GetDay()), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
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
		if err != nil || ctx.GetUserTelegramContext().GetTelegramChatId() != telegramID {
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

	t.Run("list pages by name with page_size / page_token", func(t *testing.T) {
		carol, carolTG := b.boundUser()
		for _, n := range []string{"Transport", "Food", "Bills", "Zakat", "Health"} {
			testutil.NewCategory(t, carol.ID, n)
		}

		var walked []string
		token := ""
		for page := 0; ; page++ {
			if page > 5 {
				t.Fatal("pagination did not end (is next_page_token set?)")
			}
			resp, err := b.categories.ListCategories(b.as(carolTG), &financecorev1.ListCategoriesRequest{PageSize: 2, PageToken: token})
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			for _, c := range resp.GetCategories() {
				walked = append(walked, c.GetCategoryName())
			}
			if token = resp.GetNextPageToken(); token == "" {
				break
			}
		}
		if got := strings.Join(walked, " "); got != "Bills Food Health Transport Zakat" {
			t.Fatalf("walked %q, want every category once, by name (a missing next_page_token stops after the first page)", got)
		}

		_, err := b.categories.ListCategories(b.as(carolTG), &financecorev1.ListCategoriesRequest{PageToken: "garbage!"})
		wantCode(t, err, codes.InvalidArgument, "page_token")
		_, err = b.categories.ListCategories(b.as(carolTG), &financecorev1.ListCategoriesRequest{PageSize: -1})
		wantCode(t, err, codes.InvalidArgument, "page_size")
	})

	t.Run("get returns the caller's category; another user's is NOT_FOUND", func(t *testing.T) {
		resp, err := b.categories.GetCategoryByID(b.as(aliceTG), &financecorev1.GetCategoryByIDRequest{Id: food.ID.String()})
		if err != nil || resp.GetCategory().GetId() != food.ID.String() || resp.GetCategory().GetCategoryName() != "Food" {
			t.Fatalf("got %v, %v; want Food", resp, err)
		}

		_, err = b.categories.GetCategoryByID(b.as(aliceTG), &financecorev1.GetCategoryByIDRequest{Id: bobsFood.ID.String()})
		wantCode(t, err, codes.NotFound, "")
	})

	t.Run("a malformed ID is INVALID_ARGUMENT", func(t *testing.T) {
		_, err := b.categories.GetCategoryByID(b.as(aliceTG), &financecorev1.GetCategoryByIDRequest{Id: "nope"})
		wantCode(t, err, codes.InvalidArgument, "id: must be a UUID")
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
			CategoryId: food, TransactionDt: day(t, "2026-01-15"), Description: str("  Lunch "), Amount: amount("12500.50"),
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		e := resp.GetExpense()
		if e.GetUserId() != alice.ID.String() || e.GetCategoryId() != food || e.GetCategoryName() != "Food" ||
			ymd(e.GetTransactionDt()) != "2026-01-15" || e.GetDescription() != "Lunch" || e.GetAmount().GetValue() != "12500.5" {
			t.Fatalf("got %v", e)
		}

		got, err := b.expenses.GetExpenseByID(b.as(aliceTG), &financecorev1.GetExpenseByIDRequest{Id: e.GetId()})
		if err != nil || got.GetExpense().GetCategoryName() != "Food" || got.GetExpense().GetDescription() != "Lunch" {
			t.Fatalf("get: %v, %v; want Food / Lunch", got, err)
		}
	})

	t.Run("create rejects bad input with INVALID_ARGUMENT and the field", func(t *testing.T) {
		for _, tt := range []struct {
			req     *financecorev1.CreateExpenseRequest
			message string
		}{
			{&financecorev1.CreateExpenseRequest{CategoryId: "nope", TransactionDt: day(t, "2026-01-15"), Amount: amount("1")}, "category_id: must be a UUID"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, Amount: amount("1")}, "transaction_dt: is required"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: &date.Date{Year: 2026, Month: 2, Day: 30}, Amount: amount("1")}, "transaction_dt: must be a valid calendar date"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: &date.Date{Year: 2026, Month: 13, Day: 1}, Amount: amount("1")}, "transaction_dt: must be a valid calendar date"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: &date.Date{Month: 10, Day: 17}, Amount: amount("1")}, "transaction_dt: must be a valid calendar date"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: day(t, "2026-01-15")}, "amount: is required"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: day(t, "2026-01-15"), Amount: amount("ten")}, "amount: must be a decimal number"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: day(t, "1999-12-31"), Amount: amount("1")}, "transaction_dt"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: day(t, "2026-01-15"), Amount: amount("0")}, "amount: must be greater than 0"},
			{&financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: day(t, "2026-01-15"), Amount: amount("1.005")}, "amount: at most 2 decimal places"},
			{&financecorev1.CreateExpenseRequest{CategoryId: bobsFood, TransactionDt: day(t, "2026-01-15"), Amount: amount("1")}, ""},
		} {
			_, err := b.expenses.CreateExpense(b.as(aliceTG), tt.req)
			wantCode(t, err, codes.InvalidArgument, tt.message)
		}
	})

	t.Run("update keeps an omitted description and clears it with an empty string", func(t *testing.T) {
		created, _ := b.expenses.CreateExpense(b.as(aliceTG), &financecorev1.CreateExpenseRequest{
			CategoryId: food, TransactionDt: day(t, "2026-01-15"), Description: str("Lunch"), Amount: amount("10"),
		})
		req := &financecorev1.UpdateExpenseRequest{Id: created.GetExpense().GetId(), CategoryId: &food, TransactionDt: day(t, "2026-01-16"), Amount: amount("20")}

		resp, err := b.expenses.UpdateExpense(b.as(aliceTG), req)
		kept := resp.GetExpense()
		if err != nil || kept.GetDescription() != "Lunch" || kept.GetAmount().GetValue() != "20" || ymd(kept.GetTransactionDt()) != "2026-01-16" ||
			kept.GetCategoryName() != "Food" {
			t.Fatalf("update without description: %v, %v", kept, err)
		}
		req.Description = str("")
		resp, err = b.expenses.UpdateExpense(b.as(aliceTG), req)
		cleared := resp.GetExpense()
		if err != nil || cleared.GetDescription() != "" {
			t.Fatalf("update with empty description: %v, %v", cleared, err)
		}
	})

	t.Run("update is a patch: each omitted field keeps its value", func(t *testing.T) {
		bills := testutil.NewCategory(t, alice.ID, "Bills").ID.String()
		created, err := b.expenses.CreateExpense(b.as(aliceTG), &financecorev1.CreateExpenseRequest{
			CategoryId: food, TransactionDt: day(t, "2026-01-15"), Description: str("Lunch"), Amount: amount("10"),
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		id := created.GetExpense().GetId()

		for _, step := range []struct {
			name string
			req  *financecorev1.UpdateExpenseRequest
			want string
		}{
			{"only the amount", &financecorev1.UpdateExpenseRequest{Id: id, Amount: amount("20")}, "2026-01-15 Food Lunch 20"},
			{"only the category", &financecorev1.UpdateExpenseRequest{Id: id, CategoryId: &bills}, "2026-01-15 Bills Lunch 20"},
			{"only the date", &financecorev1.UpdateExpenseRequest{Id: id, TransactionDt: day(t, "2026-01-20")}, "2026-01-20 Bills Lunch 20"},
			{"only the description", &financecorev1.UpdateExpenseRequest{Id: id, Description: str("Dinner")}, "2026-01-20 Bills Dinner 20"},
			{"nothing", &financecorev1.UpdateExpenseRequest{Id: id}, "2026-01-20 Bills Dinner 20"},
		} {
			resp, err := b.expenses.UpdateExpense(b.as(aliceTG), step.req)
			if err != nil {
				t.Fatalf("update %s: %v", step.name, err)
			}
			e := resp.GetExpense()
			if got := strings.Join([]string{ymd(e.GetTransactionDt()), e.GetCategoryName(), e.GetDescription(), e.GetAmount().GetValue()}, " "); got != step.want {
				t.Fatalf("update %s: got %q, want %q", step.name, got, step.want)
			}
		}

		_, err = b.expenses.UpdateExpense(b.as(aliceTG), &financecorev1.UpdateExpenseRequest{Id: id, Amount: amount("0")})
		wantCode(t, err, codes.InvalidArgument, "amount: must be greater than 0")
		_, err = b.expenses.UpdateExpense(b.as(aliceTG), &financecorev1.UpdateExpenseRequest{Id: id, TransactionDt: &date.Date{Year: 2026, Month: 2, Day: 30}})
		wantCode(t, err, codes.InvalidArgument, "transaction_dt: must be a valid calendar date")
	})

	t.Run("a malformed UUID is INVALID_ARGUMENT naming the field, on every RPC", func(t *testing.T) {
		bad := "nope"
		for name, call := range map[string]func() error{
			"GetExpenseByID": func() error {
				_, err := b.expenses.GetExpenseByID(b.as(aliceTG), &financecorev1.GetExpenseByIDRequest{Id: bad})
				return err
			},
			"UpdateExpense id": func() error {
				_, err := b.expenses.UpdateExpense(b.as(aliceTG), &financecorev1.UpdateExpenseRequest{Id: bad})
				return err
			},
			"DeleteExpense": func() error {
				_, err := b.expenses.DeleteExpense(b.as(aliceTG), &financecorev1.DeleteExpenseRequest{Id: bad})
				return err
			},
		} {
			t.Run(name, func(t *testing.T) { wantCode(t, call(), codes.InvalidArgument, "id: must be a UUID") })
		}

		created, err := b.expenses.CreateExpense(b.as(aliceTG), &financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: day(t, "2026-01-15"), Amount: amount("1")})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		_, err = b.expenses.UpdateExpense(b.as(aliceTG), &financecorev1.UpdateExpenseRequest{Id: created.GetExpense().GetId(), CategoryId: &bad})
		wantCode(t, err, codes.InvalidArgument, "category_id: must be a UUID")
	})

	t.Run("another user's expense is NOT_FOUND; deleting twice is NOT_FOUND", func(t *testing.T) {
		created, _ := b.expenses.CreateExpense(b.as(aliceTG), &financecorev1.CreateExpenseRequest{CategoryId: food, TransactionDt: day(t, "2026-01-15"), Amount: amount("10")})
		id := created.GetExpense().GetId()

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
			if _, err := b.expenses.CreateExpense(b.as(carolTG), &financecorev1.CreateExpenseRequest{CategoryId: carolsFood, TransactionDt: day(t, dt), Amount: amount("1")}); err != nil {
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
				dates = append(dates, ymd(e.GetTransactionDt()))
			}
			if token = resp.GetNextPageToken(); token == "" {
				break
			}
		}
		if want := "2026-02-04 2026-02-03 2026-02-02 2026-02-02 2026-02-01"; strings.Join(dates, " ") != want {
			t.Fatalf("dates %v, want %s", dates, want)
		}

		filtered, err := b.expenses.ListExpenses(b.as(carolTG), &financecorev1.ListExpensesRequest{TransactionDt: day(t, "2026-02-02")})
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
		for _, c := range []*financecorev1.UserTelegramContext{got.GetUserTelegramContext(), resp.GetUserTelegramContext()} {
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
		if err != nil || got.GetUserTelegramContext().GetTelegramChatId() != otherTG || got.GetUserTelegramContext().GetState() != "" {
			t.Fatalf("got %v, %v; want the other user's fresh context", got, err)
		}
	})
}
