package models

import (
	"context"
	"testing"

	"stren/internal/db"
)

// pushTestHarness returns a connected in-memory DB plus a
// PushSubscriptionRepository and a test user. Closing the returned
// database is the caller's responsibility (defer database.Close()).
func pushTestHarness(t *testing.T) (*PushSubscriptionRepository, *UserRepository, *db.DB, string) {
	t.Helper()

	database, err := db.NewLocalConnection(":memory:")
	if err != nil {
		t.Fatalf("in-memory db: %v", err)
	}

	userRepo := NewUserRepository(database)
	pushRepo := NewPushSubscriptionRepository(database)

	user := &User{
		Name:         "Push Test",
		Email:        "push-test@example.com",
		PasswordHash: "hash",
	}
	if err := userRepo.CreateUser(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	return pushRepo, userRepo, database, user.ID
}

func sampleSub(endpoint string) PushSubscription {
	return PushSubscription{
		Endpoint: endpoint,
		P256dh:   "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4AyfIuF8u0cKPC3tB8S5d3VbV5J8K3A8B7B4e1d2c3a4b5c6d7e8f9a0b1c2",
		Auth:     "BxSKS5R8b4wCY0Uo7w8jVA",
	}
}

func TestPushRepository_UpsertForUser_CreatesRow(t *testing.T) {
	repo, _, database, userID := pushTestHarness(t)
	defer database.Close()

	ctx := context.Background()
	row, err := repo.UpsertForUser(ctx, userID, sampleSub("https://push.example/1"))
	if err != nil {
		t.Fatalf("UpsertForUser: %v", err)
	}
	if row.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if row.Endpoint != "https://push.example/1" {
		t.Fatalf("endpoint mismatch: %q", row.Endpoint)
	}
	if row.UserID != userID {
		t.Fatalf("userID mismatch: %q", row.UserID)
	}
}

func TestPushRepository_UpsertForUser_UpdatesOnConflict(t *testing.T) {
	repo, _, database, userID := pushTestHarness(t)
	defer database.Close()

	ctx := context.Background()
	first, err := repo.UpsertForUser(ctx, userID, sampleSub("https://push.example/2"))
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Same endpoint, different keys. The row should be updated, not
	// duplicated.
	updated := sampleSub("https://push.example/2")
	updated.P256dh = "DIFFERENT-P256DH"
	updated.Auth = "DIFFERENT-AUTH"

	second, err := repo.UpsertForUser(ctx, userID, updated)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same ID after update; got %s vs %s", first.ID, second.ID)
	}
	if second.P256dh != "DIFFERENT-P256DH" {
		t.Fatalf("expected p256dh update; got %q", second.P256dh)
	}
}

func TestPushRepository_DeleteByEndpoint(t *testing.T) {
	repo, _, database, userID := pushTestHarness(t)
	defer database.Close()

	ctx := context.Background()
	_, err := repo.UpsertForUser(ctx, userID, sampleSub("https://push.example/3"))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := repo.DeleteByEndpoint(ctx, "https://push.example/3"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Idempotent: a second delete is a no-op.
	if err := repo.DeleteByEndpoint(ctx, "https://push.example/3"); err != nil {
		t.Fatalf("delete (idempotent): %v", err)
	}
}

func TestPushRepository_ListAll(t *testing.T) {
	repo, _, database, userID := pushTestHarness(t)
	defer database.Close()

	ctx := context.Background()
	for _, e := range []string{"https://push.example/a", "https://push.example/b", "https://push.example/c"} {
		if _, err := repo.UpsertForUser(ctx, userID, sampleSub(e)); err != nil {
			t.Fatalf("upsert %s: %v", e, err)
		}
	}

	rows, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestPushRepository_CountForUser(t *testing.T) {
	repo, _, database, userID := pushTestHarness(t)
	defer database.Close()

	ctx := context.Background()

	// 0 subscriptions.
	n, err := repo.CountForUser(ctx, userID)
	if err != nil {
		t.Fatalf("CountForUser: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}

	// Add 2 subscriptions.
	for _, e := range []string{"https://push.example/x", "https://push.example/y"} {
		if _, err := repo.UpsertForUser(ctx, userID, sampleSub(e)); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	n, err = repo.CountForUser(ctx, userID)
	if err != nil {
		t.Fatalf("CountForUser: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}
}

func TestPushRepository_RejectsMissingFields(t *testing.T) {
	repo, _, database, userID := pushTestHarness(t)
	defer database.Close()

	ctx := context.Background()
	bad := PushSubscription{Endpoint: "https://push.example/z", P256dh: "x"} // missing Auth
	if _, err := repo.UpsertForUser(ctx, userID, bad); err == nil {
		t.Fatal("expected error for missing auth")
	}
}
