package telegrambindingcode_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	"github.com/revirathya/revirathya-finance-core/internal/usecase/telegrambindingcode"
)

// Fake
type fakeBindCodeRepo struct {
	bindUser  *domain.User
	createErr error
	bindErr   error
	unbindErr error

	createCalls     int
	gotCreateUserID uuid.UUID
	gotCreateHash   string
	gotExpiresAt    time.Time
	bindCalls       int
	gotBindHash     string
	gotTelegramID   string
	gotUnbindUserID uuid.UUID
}

func (f *fakeBindCodeRepo) CreateBindCode(ctx context.Context, userID uuid.UUID, codeHash string, expiresAt time.Time) error {
	f.createCalls++
	f.gotCreateUserID = userID
	f.gotCreateHash = codeHash
	f.gotExpiresAt = expiresAt
	return f.createErr
}

func (f *fakeBindCodeRepo) BindCode(ctx context.Context, codeHash string, telegramID string) (*domain.User, error) {
	f.bindCalls++
	f.gotBindHash = codeHash
	f.gotTelegramID = telegramID
	return f.bindUser, f.bindErr
}

func (f *fakeBindCodeRepo) UnbindTelegram(ctx context.Context, userID uuid.UUID) error {
	f.gotUnbindUserID = userID
	return f.unbindErr
}

// Codes are shown as XXXX-XXXX, without look-alike characters (0/O, 1/I/L).
var codeFormat = regexp.MustCompile(`^[A-HJKMNP-Z2-9]{4}-[A-HJKMNP-Z2-9]{4}$`)

// Test
func TestCreateBindCode(t *testing.T) {
	userID := uuid.New()

	t.Run("returns a formatted code and stores only its hash", func(t *testing.T) {
		repo := &fakeBindCodeRepo{}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		before := time.Now()
		code, expiresAt, err := uc.CreateBindCode(context.Background(), userID)
		after := time.Now()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !codeFormat.MatchString(code) {
			t.Fatalf("code %q does not match XXXX-XXXX without look-alike characters", code)
		}
		if repo.gotCreateUserID != userID {
			t.Fatalf("repo received user %v, want %v", repo.gotCreateUserID, userID)
		}
		if len(repo.gotCreateHash) != 64 || strings.Contains(repo.gotCreateHash, strings.ReplaceAll(code, "-", "")) {
			t.Fatalf("repo received %q, want a 64-character hash, never the code itself", repo.gotCreateHash)
		}
		if expiresAt.Before(before.Add(10*time.Minute)) || expiresAt.After(after.Add(10*time.Minute)) {
			t.Fatalf("expiresAt %v is not 10 minutes from now", expiresAt)
		}
		if !repo.gotExpiresAt.Equal(expiresAt) {
			t.Fatalf("repo received expiresAt %v, returned %v", repo.gotExpiresAt, expiresAt)
		}
	})

	t.Run("stored hash matches the hash BindTelegram computes for the same code", func(t *testing.T) {
		repo := &fakeBindCodeRepo{bindUser: &domain.User{ID: userID}}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		code, _, err := uc.CreateBindCode(context.Background(), userID)
		if err != nil {
			t.Fatalf("create: unexpected error: %v", err)
		}
		if _, err := uc.BindTelegram(context.Background(), code, "123456789"); err != nil {
			t.Fatalf("bind with the generated code %q: unexpected error: %v", code, err)
		}
		if repo.gotBindHash != repo.gotCreateHash {
			t.Fatalf("bind hash %q != stored hash %q", repo.gotBindHash, repo.gotCreateHash)
		}
	})

	t.Run("generates different codes", func(t *testing.T) {
		repo := &fakeBindCodeRepo{}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		seen := make(map[string]bool)
		for range 200 {
			code, _, err := uc.CreateBindCode(context.Background(), userID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !codeFormat.MatchString(code) {
				t.Fatalf("code %q does not match XXXX-XXXX without look-alike characters", code)
			}
			seen[code] = true
		}
		if len(seen) < 195 {
			t.Fatalf("only %d distinct codes out of 200; generator is not random enough", len(seen))
		}
	})

	t.Run("propagates repo error without returning a code", func(t *testing.T) {
		repo := &fakeBindCodeRepo{createErr: errors.New("db down")}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		code, _, err := uc.CreateBindCode(context.Background(), userID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if code != "" {
			t.Fatalf("got code %q on error, want empty", code)
		}
	})
}

func TestBindTelegram(t *testing.T) {
	const canonical = "K7QX-3MPA"
	user := &domain.User{ID: uuid.New(), FirstName: "Avi"}

	canonicalHash := func(t *testing.T) string {
		t.Helper()
		repo := &fakeBindCodeRepo{bindUser: user}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)
		if _, err := uc.BindTelegram(context.Background(), canonical, "123456789"); err != nil {
			t.Fatalf("bind %q: unexpected error: %v", canonical, err)
		}
		return repo.gotBindHash
	}

	t.Run("binds and returns the user", func(t *testing.T) {
		repo := &fakeBindCodeRepo{bindUser: user}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		got, err := uc.BindTelegram(context.Background(), canonical, "123456789")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != user {
			t.Fatalf("got %v, want %v", got, user)
		}
		if repo.gotTelegramID != "123456789" {
			t.Fatalf("repo received telegram ID %q, want %q", repo.gotTelegramID, "123456789")
		}
		if len(repo.gotBindHash) != 64 {
			t.Fatalf("repo received %q, want a 64-character hash", repo.gotBindHash)
		}
	})

	t.Run("same code always gives the same hash", func(t *testing.T) {
		if first, second := canonicalHash(t), canonicalHash(t); first != second {
			t.Fatalf("hashes differ for the same code: %q vs %q", first, second)
		}
	})

	t.Run("accepts only the exact XXXX-XXXX uppercase format", func(t *testing.T) {
		for _, input := range []string{"k7qx-3mpa", "K7qx-3MPA", "K7QX3MPA", "K7QX 3MPA", " K7QX-3MPA", "K7QX-3MPA "} {
			repo := &fakeBindCodeRepo{bindUser: user}
			uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

			if _, err := uc.BindTelegram(context.Background(), input, "123456789"); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("bind %q: got %v, want domain.ErrInvalidInput", input, err)
			}
			if repo.bindCalls != 0 {
				t.Fatalf("bind %q: repo called %d times, want 0", input, repo.bindCalls)
			}
		}
	})

	t.Run("rejects malformed codes without calling the repo", func(t *testing.T) {
		for _, input := range []string{"", "K7QX", "K7QX-3MPA-X", "K7QX-3MP!", "K7QX-3MP0", "K7QX-3MPI", "K7QX-3MPL"} {
			repo := &fakeBindCodeRepo{bindUser: user}
			uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

			if _, err := uc.BindTelegram(context.Background(), input, "123456789"); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("bind %q: got %v, want domain.ErrInvalidInput", input, err)
			}
			if repo.bindCalls != 0 {
				t.Fatalf("bind %q: repo called %d times, want 0", input, repo.bindCalls)
			}
		}
	})

	t.Run("rejects empty telegram ID without calling the repo", func(t *testing.T) {
		repo := &fakeBindCodeRepo{bindUser: user}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		if _, err := uc.BindTelegram(context.Background(), canonical, ""); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("got %v, want domain.ErrInvalidInput", err)
		}
		if repo.bindCalls != 0 {
			t.Fatalf("repo called %d times, want 0", repo.bindCalls)
		}
	})

	t.Run("unknown, expired or used code is invalid input", func(t *testing.T) {
		repo := &fakeBindCodeRepo{bindErr: domain.ErrNotFound}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		if _, err := uc.BindTelegram(context.Background(), canonical, "123456789"); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("got %v, want domain.ErrInvalidInput", err)
		}
	})

	t.Run("telegram account bound to another user is a conflict", func(t *testing.T) {
		repo := &fakeBindCodeRepo{bindErr: domain.ErrConflict}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		if _, err := uc.BindTelegram(context.Background(), canonical, "123456789"); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("got %v, want domain.ErrConflict", err)
		}
	})

	t.Run("propagates other repo errors", func(t *testing.T) {
		dbErr := errors.New("db down")
		repo := &fakeBindCodeRepo{bindErr: dbErr}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		if _, err := uc.BindTelegram(context.Background(), canonical, "123456789"); !errors.Is(err, dbErr) {
			t.Fatalf("got %v, want %v", err, dbErr)
		}
	})
}

func TestUnbindTelegram(t *testing.T) {
	userID := uuid.New()

	t.Run("unbinds the user", func(t *testing.T) {
		repo := &fakeBindCodeRepo{}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		if err := uc.UnbindTelegram(context.Background(), userID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.gotUnbindUserID != userID {
			t.Fatalf("repo received user %v, want %v", repo.gotUnbindUserID, userID)
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repo := &fakeBindCodeRepo{unbindErr: errors.New("db down")}
		uc := telegrambindingcode.NewTelegramBindCodeUsecase(repo)

		if err := uc.UnbindTelegram(context.Background(), userID); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
