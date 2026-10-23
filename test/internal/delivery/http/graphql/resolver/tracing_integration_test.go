//go:build integration

package resolver_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

// spansContaining returns the spans with any attribute value containing s.
func spansContaining(spans tracetest.SpanStubs, s string) []string {
	var found []string
	for _, span := range spans {
		for _, a := range span.Attributes {
			if strings.Contains(a.Value.String(), s) {
				found = append(found, span.Name+" "+string(a.Key))
			}
		}
	}
	return found
}

func TestGraphQLTracing(t *testing.T) {
	testutil.InstallTracing(t)
	a := newAPI(t)

	t.Run("operation and resolver spans join the HTTP request's trace", func(t *testing.T) {
		token, _ := a.login("Trace", "User")
		a.ok(token, `query TraceMe { me { id } }`, nil)

		spans := testutil.Spans(t)
		byName := map[string]tracetest.SpanStub{}
		for _, s := range spans {
			byName[s.Name] = s
		}
		op, ok := byName["query TraceMe"]
		if !ok {
			t.Fatal(`no operation span "query TraceMe"`)
		}
		resolverSpan, ok := byName["Query.me"]
		if !ok {
			t.Fatal(`no resolver span "Query.me"`)
		}
		if resolverSpan.Parent.SpanID() != op.SpanContext.SpanID() {
			t.Fatal("resolver span isn't a child of the operation span")
		}
		var parentName string
		for _, s := range spans {
			if s.SpanContext.SpanID() == op.Parent.SpanID() {
				parentName = s.Name
			}
		}
		if parentName != "POST /graphql" {
			t.Fatalf("operation span's parent %q, want the HTTP span POST /graphql", parentName)
		}
	})

	t.Run("a failed operation is marked as an error", func(t *testing.T) {
		a.do("", `query TraceAnon { me { id } }`, nil) // no token: UNAUTHENTICATED

		for _, s := range testutil.Spans(t) {
			if s.Name == "query TraceAnon" {
				if s.Status.Code != codes.Error || s.Status.Description != "" {
					t.Fatalf("status %+v, want Error without a description", s.Status)
				}
				return
			}
		}
		t.Fatal(`no operation span "query TraceAnon"`)
	})

	t.Run("secrets in variables don't reach spans", func(t *testing.T) {
		secret := "token-var-" + uuid.NewString()
		a.verifier.Add(secret, domain.GoogleIdentity{Subject: uuid.NewString(), Email: uuid.NewString() + "@test.local", EmailVerified: true, GivenName: "V"})
		a.ok("", `mutation($t: String!) { loginWithGoogle(idToken: $t) { token } }`, map[string]any{"t": secret})

		if found := spansContaining(testutil.Spans(t), secret); len(found) > 0 {
			t.Fatalf("idToken recorded in %v", found)
		}
	})

	t.Run("secrets written inline in the query don't reach spans", func(t *testing.T) {
		secret := "token-inline-" + uuid.NewString()
		a.verifier.Add(secret, domain.GoogleIdentity{Subject: uuid.NewString(), Email: uuid.NewString() + "@test.local", EmailVerified: true, GivenName: "I"})
		a.ok("", `mutation { loginWithGoogle(idToken: "`+secret+`") { token } }`, nil)

		if found := spansContaining(testutil.Spans(t), secret); len(found) > 0 {
			t.Fatalf("idToken recorded in %v", found)
		}
	})
}
