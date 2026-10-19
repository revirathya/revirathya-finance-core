//go:build integration

package testutil

import (
	"context"
	"log"
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/platform/database"

	// Repository
	rCategory "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/category"
	rExpense "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/expense"
	rUser "github.com/revirathya/revirathya-finance-core/internal/repository/postgres/user"
)

var TestPool *pgxpool.Pool

// TestDSN is the connection string of the test database, e.g. for running migrations in a test.
var TestDSN string

func RunWithPostgres(m *testing.M) int {
	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres: %v", err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	TestDSN = dsn

	mig, err := migrate.New("file://"+MigrationsDir(), dsn)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := mig.Up(); err != nil {
		log.Fatalf("migrate up: %v", err)
	}

	// The production pool, so tests run with its settings (statement_timeout, ...).
	TestPool, err = database.NewDBPool(ctx, dsn)
	if err != nil {
		log.Fatalf("pool: %v", err)
	}

	defer TestPool.Close()

	return m.Run()
}

// MigrationsDir is the absolute path of migrations/.
func MigrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0) // .../testutil/postgres.go
	return filepath.Join(filepath.Dir(thisFile), "..", "migrations")
}

// Helpers
func NewUser(t *testing.T) *domain.User {
	t.Helper()

	id := uuid.NewString()
	user, err := rUser.NewUserRepositoryImpl(TestPool).CreateUser(context.Background(), domain.UserForm{
		FirstName: "Test",
		Email:     id + "@test.local",
		GoogleSub: "google-" + id,
	})
	if err != nil {
		t.Fatalf("newUser: %v", err)
	}
	return user
}

func NewCategory(t *testing.T, userID uuid.UUID, name string) *domain.Category {
	t.Helper()

	category, err := rCategory.NewCategoryRepositoryImpl(TestPool).CreateCategory(
		context.Background(), userID, domain.CategoryForm{CategoryName: name},
	)
	if err != nil {
		t.Fatalf("newCategory(%q): %v", name, err)
	}
	return category
}

// newExpense creates an expense owned by userID on date (YYYY-MM-DD) with the given amount.
func NewExpense(t *testing.T, userID, categoryID uuid.UUID, date, amount string) *domain.Expense {
	t.Helper()

	expense, err := rExpense.NewExpenseRepositoryImpl(TestPool).CreateExpense(context.Background(), domain.ExpenseForm{
		UserID:        userID,
		CategoryID:    categoryID,
		TransactionDt: Day(t, date),
		Amount:        decimal.RequireFromString(amount),
	})
	if err != nil {
		t.Fatalf("newExpense(%s, %s): %v", date, amount, err)
	}
	return expense
}

// day parses a YYYY-MM-DD date.
func Day(t *testing.T, date string) time.Time {
	t.Helper()

	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("day(%q): %v", date, err)
	}
	return d
}

// newTelegramID returns a random Telegram user ID. telegram_chat_id is UNIQUE, so
// every test needs its own.
func NewTelegramID() string {
	return strconv.FormatInt(rand.Int64N(1_000_000_000_000)+1, 10)
}

// newCodeHash returns a unique stand-in for a bind-code hash. The repository only
// stores and compares hashes, so any unique string up to 64 characters works.
func NewCodeHash() string {
	return "hash-" + uuid.NewString()
}

func Ptr[T any](v T) *T { return &v }
