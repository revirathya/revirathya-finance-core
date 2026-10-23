//go:build integration

package handler_test

import (
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Internal
	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// fakeClock is a clock the test moves forward by hand.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestWriteLimit(t *testing.T) {
	const limit = 3
	clock := &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	m := metrics.NewMetrics(nil)
	b := newBotWith(t, ratelimit.NewLimiterWithClock(limit, time.Minute, clock.now), m)

	// create sends CreateExpense for tg in category, returning the response header and the error.
	create := func(tg, category string) (metadata.MD, error) {
		var header metadata.MD
		_, err := b.expenses.CreateExpense(b.as(tg), &financecorev1.CreateExpenseRequest{
			CategoryId: category, TransactionDt: day(t, "2026-01-15"), Amount: amount("10"),
		}, grpc.Header(&header))
		return header, err
	}

	t.Run("writes past the limit are RESOURCE_EXHAUSTED with retry-after, until the window resets", func(t *testing.T) {
		user, tg := b.boundUser()
		food := testutil.NewCategory(t, user.ID, "Food").ID.String()

		for i := 1; i <= limit; i++ {
			if _, err := create(tg, food); err != nil {
				t.Fatalf("write %d: %v", i, err)
			}
		}
		clock.advance(20 * time.Second)
		header, err := create(tg, food)
		wantCode(t, err, codes.ResourceExhausted, "too many write requests, try again later")
		if got := header.Get("retry-after"); len(got) != 1 || got[0] != "40" {
			t.Fatalf("retry-after %v, want [40]", got)
		}
		if v, _ := testutil.MetricValue(t, m.Registry, "rate_limited_total", map[string]string{"api": "grpc"}); v != 1 {
			t.Fatalf("rate_limited_total{api=grpc} = %v, want 1", v)
		}

		clock.advance(40 * time.Second)
		if _, err := create(tg, food); err != nil {
			t.Fatalf("after the window: %v, want a new budget", err)
		}
	})

	t.Run("updates and deletes count too", func(t *testing.T) {
		user, tg := b.boundUser()
		food := testutil.NewCategory(t, user.ID, "Food")
		exp := testutil.NewExpense(t, user.ID, food.ID, "2026-01-15", "10")

		for i := 1; i <= limit; i++ {
			if _, err := b.expenses.UpdateExpense(b.as(tg), &financecorev1.UpdateExpenseRequest{Id: exp.ID.String(), Amount: amount("11")}); err != nil {
				t.Fatalf("update %d: %v", i, err)
			}
		}
		_, err := b.expenses.DeleteExpense(b.as(tg), &financecorev1.DeleteExpenseRequest{Id: exp.ID.String()})
		wantCode(t, err, codes.ResourceExhausted, "")
	})

	t.Run("the budget is per user", func(t *testing.T) {
		alice, aliceTG := b.boundUser()
		bob, bobTG := b.boundUser()
		aliceFood := testutil.NewCategory(t, alice.ID, "Food").ID.String()
		bobFood := testutil.NewCategory(t, bob.ID, "Food").ID.String()

		for range limit + 1 {
			_, _ = create(aliceTG, aliceFood)
		}
		if _, err := create(bobTG, bobFood); err != nil {
			t.Fatalf("bob: %v, want unaffected by alice", err)
		}
	})

	t.Run("reads and the bot's own calls don't count", func(t *testing.T) {
		user, tg := b.boundUser()
		food := testutil.NewCategory(t, user.ID, "Food").ID.String()

		for range limit + 2 {
			if _, err := b.expenses.ListExpenses(b.as(tg), &financecorev1.ListExpensesRequest{}); err != nil {
				t.Fatalf("list expenses: %v", err)
			}
			if _, err := b.categories.ListCategories(b.as(tg), &financecorev1.ListCategoriesRequest{}); err != nil {
				t.Fatalf("list categories: %v", err)
			}
			if _, err := b.contexts.UpdateUserTelegramContext(b.as(tg), &financecorev1.UpdateUserTelegramContextRequest{State: testutil.Ptr("ask_amount")}); err != nil {
				t.Fatalf("update telegram context: %v", err)
			}
		}
		for i := 1; i <= limit; i++ {
			if _, err := create(tg, food); err != nil {
				t.Fatalf("write %d after reads: %v, want the full budget left", i, err)
			}
		}
	})
}

// TestRateLimitedMethodsClassified fails when an RPC is added without deciding whether it is a counted write.
func TestRateLimitedMethodsClassified(t *testing.T) {
	notCounted := map[string]bool{
		financecorev1.CategoryService_GetCategoryByID_FullMethodName:                      true,
		financecorev1.CategoryService_ListCategories_FullMethodName:                       true,
		financecorev1.ExpenseService_GetExpenseByID_FullMethodName:                        true,
		financecorev1.ExpenseService_ListExpenses_FullMethodName:                          true,
		financecorev1.UserTelegramContextService_GetUserTelegramContext_FullMethodName:    true,
		financecorev1.UserTelegramContextService_UpdateUserTelegramContext_FullMethodName: true,
		financecorev1.UserTelegramContextService_BindTelegramAccount_FullMethodName:       true,
		"/grpc.health.v1.Health/Check":                                                    true,
		"/grpc.health.v1.Health/List":                                                     true,
		"/grpc.health.v1.Health/Watch":                                                    true,
	}
	counted := map[string]bool{}
	for _, m := range dlvGrpc.RateLimitedMethods {
		counted[m] = true
	}

	layer, err := dlvGrpc.Setup("127.0.0.1:0", false, apiKey, metrics.NewMetrics(nil), ratelimit.NewLimiter(1, time.Minute), testutil.NewUsecases(nil))
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = layer.Listener.Close() })

	for service, info := range layer.Server.GetServiceInfo() {
		for _, method := range info.Methods {
			full := "/" + service + "/" + method.Name
			if counted[full] == notCounted[full] {
				t.Errorf("%s: classify it as counted (dlvGrpc.RateLimitedMethods) or not counted (this test)", full)
			}
		}
	}
}
