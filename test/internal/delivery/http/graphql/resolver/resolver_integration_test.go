//go:build integration

package resolver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/auth"
	dlvHttp "github.com/revirathya/revirathya-finance-core/internal/delivery/http"
	dlvGql "github.com/revirathya/revirathya-finance-core/internal/delivery/http/graphql"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}

// api is the production GraphQL server (dlvHttp.Setup + RegisterHandler) on the real usecases and Postgres,
// with Google faked.
type api struct {
	t        *testing.T
	handler  http.Handler
	verifier *testutil.FakeGoogleVerifier
}

func newAPI(t *testing.T) *api {
	t.Helper()
	verifier := testutil.NewFakeGoogleVerifier()
	tokens := auth.NewTokenManager("0123456789abcdef0123456789abcdef", time.Hour)
	layer := dlvHttp.Setup("prod", ":0", func(ctx context.Context) error { return nil })
	dlvGql.RegisterHandler(layer, nil, false, tokens, metrics.NewMetrics(nil), testutil.NewUsecases(verifier))
	return &api{t: t, handler: layer.Server.Handler, verifier: verifier}
}

type result struct {
	Data   map[string]any
	Errors []struct {
		Message    string
		Extensions map[string]any
	}
}

func (r result) code() string {
	if len(r.Errors) == 0 {
		return ""
	}
	code, _ := r.Errors[0].Extensions["code"].(string)
	return code
}

// do sends one GraphQL request; token "" means anonymous.
func (a *api) do(token, query string, variables map[string]any) result {
	a.t.Helper()
	body, _ := json.Marshal(map[string]any{"query": query, "variables": variables})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	var res result
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		a.t.Fatalf("invalid response (status %d): %s", rec.Code, rec.Body.String())
	}
	return res
}

// ok sends a request and fails the test on any error.
func (a *api) ok(token, query string, variables map[string]any) map[string]any {
	a.t.Helper()
	res := a.do(token, query, variables)
	if len(res.Errors) > 0 {
		a.t.Fatalf("unexpected errors: %+v\nquery: %s", res.Errors, query)
	}
	return res.Data
}

// wantCode sends a request and fails unless the first error has code (and, if message != "", contains it).
func (a *api) wantCode(token, query string, variables map[string]any, code, message string) {
	a.t.Helper()
	res := a.do(token, query, variables)
	if res.code() != code {
		a.t.Fatalf("code %q (%+v), want %s\nquery: %s", res.code(), res.Errors, code, query)
	}
	if message != "" && !strings.Contains(res.Errors[0].Message, message) {
		a.t.Fatalf("message %q, want it to contain %q", res.Errors[0].Message, message)
	}
}

// login signs a new Google user in and returns their session token and user ID.
func (a *api) login(given, family string) (string, string) {
	a.t.Helper()
	sub := uuid.NewString()
	a.verifier.Add("token-"+sub, domain.GoogleIdentity{
		Subject: sub, Email: sub + "@test.local", EmailVerified: true, GivenName: given, FamilyName: family,
	})
	data := a.ok("", `mutation($t: String!) { loginWithGoogle(idToken: $t) { token user { id } } }`, map[string]any{"t": "token-" + sub})
	payload := data["loginWithGoogle"].(map[string]any)
	return payload["token"].(string), payload["user"].(map[string]any)["id"].(string)
}

// field walks a decoded response: field(data, "createExpense", "id").
func field(v any, path ...string) any {
	for _, p := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[p]
	}
	return v
}

func TestAuthResolvers(t *testing.T) {
	a := newAPI(t)
	const loginQuery = `mutation($t: String!) { loginWithGoogle(idToken: $t) { token expiresAt user { id firstName lastName email } } }`

	t.Run("first login creates the user from the Google profile; the next one finds it", func(t *testing.T) {
		a.verifier.Add("tok-avi", domain.GoogleIdentity{Subject: "sub-avi", Email: "Avi@Test.local", EmailVerified: true, GivenName: "Avi", FamilyName: "Leonardi"})

		first := a.ok("", loginQuery, map[string]any{"t": "tok-avi"})
		second := a.ok("", loginQuery, map[string]any{"t": "tok-avi"})

		user := field(first, "loginWithGoogle", "user").(map[string]any)
		if user["firstName"] != "Avi" || user["lastName"] != "Leonardi" || user["email"] != "avi@test.local" {
			t.Fatalf("user %v, want Avi Leonardi with the email lowercased", user)
		}
		if field(second, "loginWithGoogle", "user", "id") != user["id"] {
			t.Fatal("second login created another user")
		}
		if _, err := time.Parse(time.RFC3339, field(first, "loginWithGoogle", "expiresAt").(string)); err != nil {
			t.Fatalf("expiresAt is not RFC 3339: %v", err)
		}
	})

	t.Run("missing given name falls back to the email's local part, and no family name is null", func(t *testing.T) {
		a.verifier.Add("tok-noname", domain.GoogleIdentity{Subject: "sub-noname", Email: "jdoe@test.local", EmailVerified: true})

		user := field(a.ok("", loginQuery, map[string]any{"t": "tok-noname"}), "loginWithGoogle", "user").(map[string]any)
		if user["firstName"] != "jdoe" || user["lastName"] != nil {
			t.Fatalf("user %v, want firstName jdoe and lastName null", user)
		}
	})

	t.Run("unknown token and unverified email are UNAUTHENTICATED", func(t *testing.T) {
		a.verifier.Add("tok-unverified", domain.GoogleIdentity{Subject: "sub-unverified", Email: "x@test.local", GivenName: "X"})

		a.wantCode("", loginQuery, map[string]any{"t": "garbage"}, "UNAUTHENTICATED", "")
		a.wantCode("", loginQuery, map[string]any{"t": "tok-unverified"}, "UNAUTHENTICATED", "")
	})

	t.Run("the session token authenticates later requests", func(t *testing.T) {
		token, id := a.login("Budi", "")

		if got := field(a.ok(token, `{ me { id firstName } }`, nil), "me", "id"); got != id {
			t.Fatalf("me.id %v, want %s", got, id)
		}
		a.wantCode("", `{ me { id } }`, nil, "UNAUTHENTICATED", "")
		a.wantCode("not-a-jwt", `{ me { id } }`, nil, "UNAUTHENTICATED", "")
	})

	t.Run("updateUser trims, clears the last name with an empty string, and validates", func(t *testing.T) {
		token, _ := a.login("Citra", "Dewi")
		const q = `mutation($in: UpdateUserInput!) { updateUser(patch: $in) { firstName lastName } }`

		user := field(a.ok(token, q, map[string]any{"in": map[string]any{"firstName": "  Citra Ayu ", "lastName": ""}}), "updateUser").(map[string]any)
		if user["firstName"] != "Citra Ayu" || user["lastName"] != nil {
			t.Fatalf("user %v, want trimmed first name and no last name", user)
		}
		blank := field(a.ok(token, q, map[string]any{"in": map[string]any{"firstName": "Citra", "lastName": "   "}}), "updateUser", "lastName")
		if blank != nil {
			t.Fatalf("whitespace-only last name stored as %q, want null", blank)
		}
		a.wantCode(token, q, map[string]any{"in": map[string]any{"firstName": "   "}}, "BAD_USER_INPUT", "first_name: is required")

		// Patch: omitted fields keep their value.
		a.ok(token, q, map[string]any{"in": map[string]any{"firstName": "Citra", "lastName": "Dewi"}})
		onlyFirst := field(a.ok(token, q, map[string]any{"in": map[string]any{"firstName": "Ayu"}}), "updateUser").(map[string]any)
		if onlyFirst["firstName"] != "Ayu" || onlyFirst["lastName"] != "Dewi" {
			t.Fatalf("user %v after a first-name-only update, want last name kept (Dewi)", onlyFirst)
		}
		onlyLast := field(a.ok(token, q, map[string]any{"in": map[string]any{"lastName": "Putri"}}), "updateUser").(map[string]any)
		if onlyLast["firstName"] != "Ayu" || onlyLast["lastName"] != "Putri" {
			t.Fatalf("user %v after a last-name-only update, want first name kept (Ayu)", onlyLast)
		}
		nulls := field(a.ok(token, q, map[string]any{"in": map[string]any{"firstName": nil, "lastName": nil}}), "updateUser").(map[string]any)
		if nulls["firstName"] != "Ayu" || nulls["lastName"] != "Putri" {
			t.Fatalf("user %v after an update with explicit nulls, want both kept", nulls)
		}
		a.wantCode(token, q, map[string]any{"in": map[string]any{"firstName": strings.Repeat("a", 101)}}, "BAD_USER_INPUT", "first_name: must be at most 100 characters")
	})
}

func TestCategoryResolvers(t *testing.T) {
	a := newAPI(t)
	alice, _ := a.login("Alice", "")
	bob, _ := a.login("Bob", "")
	const create = `mutation($n: String!) { createCategory(form: { categoryName: $n }) { id categoryName } }`

	food := field(a.ok(alice, create, map[string]any{"n": "  Food "}), "createCategory").(map[string]any)
	a.ok(alice, create, map[string]any{"n": "Bills"})
	a.ok(bob, create, map[string]any{"n": "Food"}) // same name, other user: allowed

	t.Run("create trims the name, and a duplicate for the same user is CONFLICT", func(t *testing.T) {
		if food["categoryName"] != "Food" {
			t.Fatalf("name %v, want trimmed Food", food["categoryName"])
		}
		a.wantCode(alice, create, map[string]any{"n": "Food"}, "CONFLICT", "")
		a.wantCode(alice, create, map[string]any{"n": "  "}, "BAD_USER_INPUT", "category_name: is required")
	})

	t.Run("list returns only the caller's categories, by name", func(t *testing.T) {
		list := field(a.ok(alice, `{ categories { nodes { categoryName } } }`, nil), "categories", "nodes").([]any)
		if len(list) != 2 || field(list[0], "categoryName") != "Bills" || field(list[1], "categoryName") != "Food" {
			t.Fatalf("categories %v, want alice's Bills then Food", list)
		}
	})

	t.Run("categories pages by name with first/after", func(t *testing.T) {
		dewi, _ := a.login("Dewi", "")
		for _, n := range []string{"Transport", "Food", "Bills", "Zakat", "Health"} {
			a.ok(dewi, create, map[string]any{"n": n})
		}
		const list = `query($after: String) { categories(first: 2, after: $after) { nodes { categoryName } pageInfo { hasNextPage endCursor } } }`

		var walked []string
		var after any
		for page := 0; ; page++ {
			if page > 5 {
				t.Fatal("pagination did not end")
			}
			conn := field(a.ok(dewi, list, map[string]any{"after": after}), "categories")
			for _, n := range field(conn, "nodes").([]any) {
				walked = append(walked, field(n, "categoryName").(string))
			}
			if field(conn, "pageInfo", "hasNextPage") != true {
				if field(conn, "pageInfo", "endCursor") != nil {
					t.Fatal("last page has an endCursor")
				}
				break
			}
			after = field(conn, "pageInfo", "endCursor")
		}
		if got := strings.Join(walked, " "); got != "Bills Food Health Transport Zakat" {
			t.Fatalf("walked %q, want every category once, by name", got)
		}
		a.wantCode(dewi, list, map[string]any{"after": "garbage!"}, "BAD_USER_INPUT", "page_token")
		a.wantCode(dewi, `{ categories(first: -1) { nodes { id } } }`, nil, "BAD_USER_INPUT", "page_size")
		// A token that decodes to bytes that aren't UTF-8 must be bad input, not a database error.
		a.wantCode(dewi, list, map[string]any{"after": "__4"}, "BAD_USER_INPUT", "page_token")
	})

	t.Run("another user's category is NOT_FOUND for get, update and delete", func(t *testing.T) {
		id := map[string]any{"id": food["id"]}
		if got := a.do(bob, `query($id: UUID!) { category(id: $id) { id } }`, id); got.code() != "NOT_FOUND" {
			t.Fatalf("get: code %q, want NOT_FOUND", got.code())
		}
		a.wantCode(bob, `mutation($id: UUID!) { updateCategory(id: $id, patch: { categoryName: "Hacked" }) { id } }`, id, "NOT_FOUND", "")
		a.wantCode(bob, `mutation($id: UUID!) { deleteCategory(id: $id) }`, id, "NOT_FOUND", "")
	})

	t.Run("update renames; a category in use can't be deleted; an unused one can", func(t *testing.T) {
		renamed := a.ok(alice, `mutation($id: UUID!) { updateCategory(id: $id, patch: { categoryName: "Groceries" }) { categoryName } }`, map[string]any{"id": food["id"]})
		if field(renamed, "updateCategory", "categoryName") != "Groceries" {
			t.Fatalf("renamed %v, want Groceries", renamed)
		}

		a.ok(alice, `mutation($c: UUID!) { createExpense(form: { transactionDt: "2026-01-15", categoryId: $c, amount: "10" }) { id } }`, map[string]any{"c": food["id"]})
		a.wantCode(alice, `mutation($id: UUID!) { deleteCategory(id: $id) }`, map[string]any{"id": food["id"]}, "CONFLICT", "")

		unused := field(a.ok(alice, create, map[string]any{"n": "Unused"}), "createCategory", "id")
		if got := field(a.ok(alice, `mutation($id: UUID!) { deleteCategory(id: $id) }`, map[string]any{"id": unused}), "deleteCategory"); got != unused {
			t.Fatalf("deleteCategory returned %v, want the deleted id %v", got, unused)
		}
	})
}

func TestExpenseResolvers(t *testing.T) {
	a := newAPI(t)
	alice, aliceID := a.login("Alice", "")
	bob, _ := a.login("Bob", "")
	category := func(token, name string) string {
		return field(a.ok(token, `mutation($n: String!) { createCategory(form: { categoryName: $n }) { id } }`, map[string]any{"n": name}), "createCategory", "id").(string)
	}
	food, bobsFood := category(alice, "Food"), category(bob, "Food")
	const create = `mutation($in: CreateExpenseInput!) { createExpense(form: $in) { id userId categoryId categoryName transactionDt description amount } }`
	expenseInput := func(dt, amount string, extra map[string]any) map[string]any {
		in := map[string]any{"transactionDt": dt, "categoryId": food, "amount": amount}
		for k, v := range extra {
			in[k] = v
		}
		return map[string]any{"in": in}
	}

	t.Run("create returns every field; a missing description is an empty string", func(t *testing.T) {
		e := field(a.ok(alice, create, expenseInput("2026-01-15", "12500.50", nil)), "createExpense").(map[string]any)

		if e["userId"] != aliceID || e["categoryId"] != food || e["categoryName"] != "Food" ||
			e["transactionDt"] != "2026-01-15" || e["amount"] != "12500.5" || e["description"] != "" {
			t.Fatalf("expense %v", e)
		}
	})

	t.Run("create validates through the usecase and reports the field", func(t *testing.T) {
		a.wantCode(alice, create, expenseInput("2026-01-15", "0", nil), "BAD_USER_INPUT", "amount: must be greater than 0")
		a.wantCode(alice, create, expenseInput("1999-12-31", "1", nil), "BAD_USER_INPUT", "transaction_dt")
		a.wantCode(alice, create, expenseInput("2026-01-15", "1", map[string]any{"description": strings.Repeat("a", 501)}), "BAD_USER_INPUT", "description")
		a.wantCode(alice, create, expenseInput("2026-01-15", "1", map[string]any{"categoryId": bobsFood}), "BAD_USER_INPUT", "")
	})

	t.Run("update keeps an omitted description and clears it with an empty string", func(t *testing.T) {
		id := field(a.ok(alice, create, expenseInput("2026-01-15", "10", map[string]any{"description": "Lunch"})), "createExpense", "id")
		const update = `mutation($id: UUID!, $p: UpdateExpenseInput!) { updateExpense(id: $id, patch: $p) { description amount } }`
		patch := map[string]any{"transactionDt": "2026-01-16", "categoryId": food, "amount": "20"}

		kept := field(a.ok(alice, update, map[string]any{"id": id, "p": patch}), "updateExpense").(map[string]any)
		patch["description"] = ""
		cleared := field(a.ok(alice, update, map[string]any{"id": id, "p": patch}), "updateExpense").(map[string]any)

		if kept["description"] != "Lunch" || kept["amount"] != "20" || cleared["description"] != "" {
			t.Fatalf("kept %v, cleared %v; want Lunch then empty", kept, cleared)
		}
	})

	t.Run("update is a patch: each omitted field keeps its value", func(t *testing.T) {
		bills := category(alice, "Groceries")
		id := field(a.ok(alice, create, expenseInput("2026-01-15", "10", map[string]any{"description": "Lunch"})), "createExpense", "id")
		const update = `mutation($id: UUID!, $p: UpdateExpenseInput!) { updateExpense(id: $id, patch: $p) { transactionDt categoryName description amount } }`

		for _, step := range []struct {
			patch map[string]any
			want  string
		}{
			{map[string]any{"amount": "20"}, "2026-01-15 Food Lunch 20"},
			{map[string]any{"categoryId": bills}, "2026-01-15 Groceries Lunch 20"},
			{map[string]any{"transactionDt": "2026-01-20"}, "2026-01-20 Groceries Lunch 20"},
			{map[string]any{"description": "Dinner"}, "2026-01-20 Groceries Dinner 20"},
			{map[string]any{}, "2026-01-20 Groceries Dinner 20"},
		} {
			e := field(a.ok(alice, update, map[string]any{"id": id, "p": step.patch}), "updateExpense").(map[string]any)
			if got := strings.Join([]string{e["transactionDt"].(string), e["categoryName"].(string), e["description"].(string), e["amount"].(string)}, " "); got != step.want {
				t.Fatalf("after patch %v: got %q, want %q", step.patch, got, step.want)
			}
		}
		a.wantCode(alice, update, map[string]any{"id": id, "p": map[string]any{"amount": "0"}}, "BAD_USER_INPUT", "amount: must be greater than 0")
	})

	t.Run("updateCategory without a name keeps it", func(t *testing.T) {
		id := category(alice, "Transport")
		got := field(a.ok(alice, `mutation($id: UUID!) { updateCategory(id: $id, patch: {}) { categoryName } }`, map[string]any{"id": id}), "updateCategory", "categoryName")
		if got != "Transport" {
			t.Fatalf("categoryName %v, want Transport unchanged", got)
		}
	})

	t.Run("update returns the new category's id and name", func(t *testing.T) {
		bills := category(alice, "Bills")
		id := field(a.ok(alice, create, expenseInput("2026-01-15", "10", nil)), "createExpense", "id")

		e := field(a.ok(alice, `mutation($id: UUID!, $p: UpdateExpenseInput!) { updateExpense(id: $id, patch: $p) { categoryId categoryName } }`,
			map[string]any{"id": id, "p": map[string]any{"transactionDt": "2026-01-15", "categoryId": bills, "amount": "10"}}), "updateExpense").(map[string]any)
		if e["categoryId"] != bills || e["categoryName"] != "Bills" {
			t.Fatalf("expense %v, want categoryId %s and categoryName Bills", e, bills)
		}
	})

	t.Run("another user's expense is NOT_FOUND; deleting twice is NOT_FOUND", func(t *testing.T) {
		id := field(a.ok(alice, create, expenseInput("2026-01-15", "10", nil)), "createExpense", "id")
		vars := map[string]any{"id": id}

		if got := a.do(bob, `query($id: UUID!) { expense(id: $id) { id } }`, vars); got.code() != "NOT_FOUND" {
			t.Fatalf("get: code %q, want NOT_FOUND", got.code())
		}
		a.wantCode(bob, `mutation($id: UUID!) { deleteExpense(id: $id) }`, vars, "NOT_FOUND", "")
		a.ok(alice, `mutation($id: UUID!) { deleteExpense(id: $id) }`, vars)
		a.wantCode(alice, `mutation($id: UUID!) { deleteExpense(id: $id) }`, vars, "NOT_FOUND", "")
	})

	t.Run("expenses pages through the caller's expenses with first/after", func(t *testing.T) {
		carol, _ := a.login("Carol", "")
		carolsFood := category(carol, "Food")
		for _, dt := range []string{"2026-02-01", "2026-02-02", "2026-02-02", "2026-02-03", "2026-02-04"} {
			a.ok(carol, create, map[string]any{"in": map[string]any{"transactionDt": dt, "categoryId": carolsFood, "amount": "1"}})
		}
		const list = `query($after: String) { expenses(first: 2, after: $after) { nodes { id transactionDt } pageInfo { hasNextPage endCursor } } }`

		var dates []string
		seen := map[any]bool{}
		var after any
		for page := 0; ; page++ {
			if page > 5 {
				t.Fatal("pagination did not end")
			}
			conn := field(a.ok(carol, list, map[string]any{"after": after}), "expenses")
			for _, n := range field(conn, "nodes").([]any) {
				if seen[field(n, "id")] {
					t.Fatalf("expense %v returned twice", field(n, "id"))
				}
				seen[field(n, "id")] = true
				dates = append(dates, field(n, "transactionDt").(string))
			}
			if field(conn, "pageInfo", "hasNextPage") != true {
				if field(conn, "pageInfo", "endCursor") != nil {
					t.Fatal("last page has an endCursor")
				}
				break
			}
			after = field(conn, "pageInfo", "endCursor")
		}

		want := "2026-02-04 2026-02-03 2026-02-02 2026-02-02 2026-02-01"
		if strings.Join(dates, " ") != want {
			t.Fatalf("dates %v, want %s (newest first, each once)", dates, want)
		}
		a.wantCode(carol, list, map[string]any{"after": "garbage!"}, "BAD_USER_INPUT", "page_token")
	})

	t.Run("expenses filters by date", func(t *testing.T) {
		dave, _ := a.login("Dave", "")
		davesFood := category(dave, "Food")
		for _, dt := range []string{"2026-03-01", "2026-03-02", "2026-03-02"} {
			a.ok(dave, create, map[string]any{"in": map[string]any{"transactionDt": dt, "categoryId": davesFood, "amount": "1"}})
		}

		nodes := field(a.ok(dave, `{ expenses(transactionDt: "2026-03-02") { nodes { transactionDt } } }`, nil), "expenses", "nodes").([]any)
		if len(nodes) != 2 {
			t.Fatalf("got %d expenses, want the 2 on 2026-03-02", len(nodes))
		}
	})
}

func TestTelegramResolvers(t *testing.T) {
	a := newAPI(t)
	token, _ := a.login("Eka", "")

	t.Run("createTelegramBindCode returns an XXXX-XXXX code valid for about 10 minutes", func(t *testing.T) {
		bind := field(a.ok(token, `mutation { createTelegramBindCode { code expiresAt } }`, nil), "createTelegramBindCode").(map[string]any)

		if !auth.ValidateCode(bind["code"].(string)) {
			t.Fatalf("code %q is not in XXXX-XXXX format", bind["code"])
		}
		expires, err := time.Parse(time.RFC3339, bind["expiresAt"].(string))
		if err != nil || time.Until(expires) < 9*time.Minute || time.Until(expires) > 11*time.Minute {
			t.Fatalf("expiresAt %v (%v), want about 10 minutes from now", bind["expiresAt"], err)
		}
	})

	t.Run("unbindTelegram returns the user without a Telegram chat", func(t *testing.T) {
		user := field(a.ok(token, `mutation { unbindTelegram { id telegramChatId } }`, nil), "unbindTelegram").(map[string]any)
		if user["telegramChatId"] != nil {
			t.Fatalf("telegramChatId %v, want null", user["telegramChatId"])
		}
	})

	t.Run("both need a session", func(t *testing.T) {
		a.wantCode("", `mutation { createTelegramBindCode { code } }`, nil, "UNAUTHENTICATED", "")
		a.wantCode("", `mutation { unbindTelegram { id } }`, nil, "UNAUTHENTICATED", "")
	})
}
