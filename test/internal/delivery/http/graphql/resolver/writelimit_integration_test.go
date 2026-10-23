//go:build integration

package resolver_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/type/date"
	gdecimal "google.golang.org/genproto/googleapis/type/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	// Generated
	schema "github.com/revirathya/revirathya-finance-core/gen/graphql"
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Internal
	dlvGrpc "github.com/revirathya/revirathya-finance-core/internal/delivery/grpc"
	dlvGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/internal/platform/ratelimit"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

const createCategory = `mutation($f: CreateCategoryInput!) { createCategory(form: $f) { id } }`

func categoryVars() map[string]any {
	return map[string]any{"f": map[string]any{"categoryName": "c-" + uuid.NewString()}}
}

func TestWriteLimit(t *testing.T) {
	const limit = 3

	t.Run("mutations past the limit are RATE_LIMITED with retryAfter", func(t *testing.T) {
		m := metrics.NewMetrics(nil)
		a := newAPIWith(t, ratelimit.NewLimiter(limit, time.Minute), m)
		token, _ := a.login("Rate", "Limited")

		for i := 1; i <= limit; i++ {
			a.ok(token, createCategory, categoryVars())
		}
		res := a.do(token, createCategory, categoryVars())

		if res.code() != "RATE_LIMITED" || res.Errors[0].Message != "too many write requests, try again later" {
			t.Fatalf("errors %+v, want RATE_LIMITED", res.Errors)
		}
		if retry, _ := res.Errors[0].Extensions["retryAfter"].(float64); retry < 1 || retry > 60 {
			t.Fatalf("retryAfter %v, want 1-60 seconds", res.Errors[0].Extensions["retryAfter"])
		}
		if v, _ := testutil.MetricValue(t, m.Registry, "rate_limited_total", map[string]string{"api": "graphql"}); v != 1 {
			t.Fatalf("rate_limited_total{api=graphql} = %v, want 1", v)
		}
	})

	t.Run("each mutation field in a document counts; the ones past the limit fail on their own", func(t *testing.T) {
		a := newAPIWith(t, ratelimit.NewLimiter(limit, time.Minute), metrics.NewMetrics(nil))
		token, _ := a.login("Multi", "Mutation")
		a.ok(token, createCategory, categoryVars()) // 2 writes left

		res := a.do(token, `mutation {
			a: createCategory(form: {categoryName: "a"}) { id }
			b: createCategory(form: {categoryName: "b"}) { id }
			c: createCategory(form: {categoryName: "c"}) { id }
		}`, nil)

		if len(res.Errors) != 1 || res.code() != "RATE_LIMITED" {
			t.Fatalf("errors %+v, want one RATE_LIMITED", res.Errors)
		}
		if res.Data["a"] == nil || res.Data["b"] == nil || res.Data["c"] != nil {
			t.Fatalf("data %v, want a and b created, c null", res.Data)
		}
	})

	t.Run("the budget is per user", func(t *testing.T) {
		a := newAPIWith(t, ratelimit.NewLimiter(limit, time.Minute), metrics.NewMetrics(nil))
		alice, _ := a.login("Alice", "A")
		bob, _ := a.login("Bob", "B")

		for range limit + 1 {
			a.do(alice, createCategory, categoryVars())
		}
		a.ok(bob, createCategory, categoryVars())
	})

	t.Run("queries, loginWithGoogle and anonymous mutations don't count", func(t *testing.T) {
		a := newAPIWith(t, ratelimit.NewLimiter(limit, time.Minute), metrics.NewMetrics(nil))
		sub := uuid.NewString()
		a.verifier.Add("token-"+sub, domain.GoogleIdentity{Subject: sub, Email: sub + "@test.local", EmailVerified: true, GivenName: "R"})
		var token string
		for range limit + 1 {
			data := a.ok("", `mutation($t: String!) { loginWithGoogle(idToken: $t) { token } }`, map[string]any{"t": "token-" + sub})
			token = field(data, "loginWithGoogle", "token").(string)
			a.ok(token, `query { me { id } categories { nodes { id } } }`, nil)
			if res := a.do("", createCategory, categoryVars()); res.code() != "UNAUTHENTICATED" {
				t.Fatalf("anonymous mutation: %+v, want UNAUTHENTICATED", res.Errors)
			}
		}
		for i := 1; i <= limit; i++ {
			a.ok(token, createCategory, categoryVars())
		}
	})

	t.Run("the budget is shared with gRPC", func(t *testing.T) {
		limiter := ratelimit.NewLimiter(limit, time.Minute)
		a := newAPIWith(t, limiter, metrics.NewMetrics(nil))
		token, userID := a.login("Shared", "Budget")

		uc := testutil.NewUsecases(testutil.NewFakeGoogleVerifier())
		layer, err := dlvGrpc.Setup("127.0.0.1:0", false, "test-api-key", metrics.NewMetrics(nil), limiter, uc)
		if err != nil {
			t.Fatalf("grpc setup: %v", err)
		}
		go func() { _ = layer.Server.Serve(layer.Listener) }()
		t.Cleanup(layer.Server.Stop)
		conn, err := grpc.NewClient(layer.Listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("grpc client: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tg := testutil.NewTelegramID()
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer test-api-key", "x-telegram-user-id", tg)
		code, _, err := uc.TelegramBindCode.CreateBindCode(ctx, uuid.MustParse(userID)) // usecase directly: not counted
		if err != nil {
			t.Fatalf("bind code: %v", err)
		}
		if _, err := financecorev1.NewUserTelegramContextServiceClient(conn).BindTelegramAccount(ctx, &financecorev1.BindTelegramAccountRequest{Code: code}); err != nil {
			t.Fatalf("bind: %v", err)
		}

		category := field(a.ok(token, createCategory, categoryVars()), "createCategory", "id").(string) // GraphQL: 1
		expenses := financecorev1.NewExpenseServiceClient(conn)
		req := &financecorev1.CreateExpenseRequest{CategoryId: category, TransactionDt: &date.Date{Year: 2026, Month: 1, Day: 15}, Amount: &gdecimal.Decimal{Value: "10"}}
		for i := 2; i <= limit; i++ { // gRPC: 2, 3
			if _, err := expenses.CreateExpense(ctx, req); err != nil {
				t.Fatalf("grpc write %d: %v", i, err)
			}
		}

		if res := a.do(token, createCategory, categoryVars()); res.code() != "RATE_LIMITED" {
			t.Fatalf("GraphQL after gRPC used the budget: %+v, want RATE_LIMITED", res.Errors)
		}
		if _, err := expenses.CreateExpense(ctx, req); status.Code(err) != codes.ResourceExhausted {
			t.Fatalf("gRPC: %v, want ResourceExhausted", err)
		}
	})
}

// TestRateLimitedMutationsClassified fails when a mutation is added without deciding whether it is a counted write.
func TestRateLimitedMutationsClassified(t *testing.T) {
	notCounted := map[string]bool{"loginWithGoogle": true, "ok": true}
	counted := map[string]bool{}
	for _, f := range dlvGql.RateLimitedMutations {
		counted[f] = true
	}

	for _, f := range schema.NewExecutableSchema(schema.Config{}).Schema().Mutation.Fields {
		if f.Name == "__typename" || f.Name == "__schema" || f.Name == "__type" {
			continue
		}
		if counted[f.Name] == notCounted[f.Name] {
			t.Errorf("%s: classify it as counted (dlvGql.RateLimitedMutations) or not counted (this test)", f.Name)
		}
	}
}
