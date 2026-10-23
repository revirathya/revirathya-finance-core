//go:build integration

package handler_test

import (
	"testing"

	// Generated
	financecorev1 "github.com/revirathya/revirathya-finance-core/gen/grpc/financecore/v1"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestAccessLogUserID(t *testing.T) {
	b := newBot(t)
	user, tg := b.boundUser()
	logs := testutil.CaptureLogs(t)

	if _, err := b.categories.ListCategories(b.as(tg), &financecorev1.ListCategoriesRequest{}); err != nil {
		t.Fatalf("list categories: %v", err)
	}

	for _, line := range testutil.LogLines(t, logs) {
		if line["msg"] == "grpc request" && line["method"] == financecorev1.CategoryService_ListCategories_FullMethodName {
			if line["user_id"] != user.ID.String() {
				t.Fatalf("user_id %v, want %s", line["user_id"], user.ID)
			}
			return
		}
	}
	t.Fatal("no access log line for ListCategories")
}
