//go:build integration

package migrations_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// Test Util
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithPostgres(m))
}

// 000005 makes expenses.description NOT NULL DEFAULT ”. Existing rows must survive both directions.
func TestExpenseDescriptionNotNullMigration(t *testing.T) {
	ctx := context.Background()
	mig, err := migrate.New("file://"+testutil.MigrationsDir(), testutil.TestDSN)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		// Leave the schema at the latest version for any other test in this package.
		if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			t.Errorf("migrate up in cleanup: %v", err)
		}
	})

	alice := testutil.NewUser(t)
	food := testutil.NewCategory(t, alice.ID, "Food")

	insert := func(t *testing.T, description any) string {
		t.Helper()
		var id string
		err := testutil.TestPool.QueryRow(ctx,
			`INSERT INTO app.expenses (user_id, category_id, transaction_dt, description, amount)
			 VALUES ($1, $2, '2026-01-15', $3, 10) RETURNING id::text`,
			alice.ID, food.ID, description).Scan(&id)
		if err != nil {
			t.Fatalf("insert expense: %v", err)
		}
		return id
	}
	description := func(t *testing.T, id string) *string {
		t.Helper()
		var d *string
		if err := testutil.TestPool.QueryRow(ctx, "SELECT description FROM app.expenses WHERE id = $1", id).Scan(&d); err != nil {
			t.Fatalf("select description: %v", err)
		}
		return d
	}

	// Back to 000004, where description is nullable.
	if err := mig.Migrate(4); err != nil {
		t.Fatalf("migrate down to 4: %v", err)
	}
	withNull := insert(t, nil)
	withText := insert(t, "Lunch")

	t.Run("up turns NULL into an empty string and keeps text", func(t *testing.T) {
		if err := mig.Migrate(5); err != nil {
			t.Fatalf("migrate up to 5: %v", err)
		}
		if d := description(t, withNull); d == nil || *d != "" {
			t.Fatalf("former NULL is %v, want empty string", d)
		}
		if d := description(t, withText); d == nil || *d != "Lunch" {
			t.Fatalf("text is %v, want Lunch", d)
		}
	})

	t.Run("up sets the column default to an empty string", func(t *testing.T) {
		var d string
		err := testutil.TestPool.QueryRow(ctx,
			`INSERT INTO app.expenses (user_id, category_id, transaction_dt, amount)
			 VALUES ($1, $2, '2026-01-15', 10) RETURNING description`,
			alice.ID, food.ID).Scan(&d)
		if err != nil || d != "" {
			t.Fatalf("got %q, %v; want empty string", d, err)
		}
	})

	t.Run("down turns empty strings back into NULL and keeps text", func(t *testing.T) {
		if err := mig.Migrate(4); err != nil {
			t.Fatalf("migrate down to 4: %v", err)
		}
		if d := description(t, withNull); d != nil {
			t.Fatalf("empty description is %q after down, want NULL", *d)
		}
		if d := description(t, withText); d == nil || *d != "Lunch" {
			t.Fatalf("text is %v after down, want Lunch", d)
		}
		insert(t, nil) // nullable again
	})
}
