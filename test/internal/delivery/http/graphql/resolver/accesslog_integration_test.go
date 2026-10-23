//go:build integration

package resolver_test

import (
	"testing"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestAccessLogUserID(t *testing.T) {
	a := newAPI(t)
	token, userID := a.login("Log", "Line")
	logs := testutil.CaptureLogs(t)

	a.ok(token, `query { me { id } }`, nil)
	a.do("", `query { me { id } }`, nil)

	var withUser, anonymous int
	for _, line := range testutil.LogLines(t, logs) {
		if line["msg"] != "http request" {
			continue
		}
		if line["user_id"] == userID {
			withUser++
		} else if _, ok := line["user_id"]; !ok {
			anonymous++
		}
	}
	if withUser != 1 || anonymous != 1 {
		t.Fatalf("access lines with user_id %d, without %d; want 1 and 1", withUser, anonymous)
	}
}
