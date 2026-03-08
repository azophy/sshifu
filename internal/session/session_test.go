package session

import (
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store := NewStore(5 * time.Minute)
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	if store.sessions == nil {
		t.Error("sessions map not initialized")
	}
	if store.maxAge != 5*time.Minute {
		t.Errorf("expected maxAge 5m, got %v", store.maxAge)
	}
}

func TestNewStoreDefaultMaxAge(t *testing.T) {
	store := NewStore(0)
	if store.maxAge != 15*time.Minute {
		t.Errorf("expected default maxAge 15m, got %v", store.maxAge)
	}
}

func TestCreateAndGet(t *testing.T) {
	store := NewStore(5 * time.Minute)

	session := store.Create("test-session-1")
	if session == nil {
		t.Fatal("Create returned nil")
	}
	if session.ID != "test-session-1" {
		t.Errorf("expected ID test-session-1, got %s", session.ID)
	}
	if session.Status != StatusPending {
		t.Errorf("expected status pending, got %s", session.Status)
	}

	got, exists := store.Get("test-session-1")
	if !exists {
		t.Error("Get returned false for existing session")
	}
	if got.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, got.ID)
	}
}

func TestGetNonExistent(t *testing.T) {
	store := NewStore(5 * time.Minute)

	_, exists := store.Get("non-existent")
	if exists {
		t.Error("Get returned true for non-existent session")
	}
}

func TestApprove(t *testing.T) {
	store := NewStore(5 * time.Minute)

	store.Create("test-session-2")

	success := store.Approve("test-session-2", "testuser", "token123")
	if !success {
		t.Error("Approve returned false")
	}

	session, _ := store.Get("test-session-2")
	if session.Status != StatusApproved {
		t.Errorf("expected status approved, got %s", session.Status)
	}
	if session.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", session.Username)
	}
	if session.AccessToken != "token123" {
		t.Errorf("expected token token123, got %s", session.AccessToken)
	}
}

func TestApproveNonExistent(t *testing.T) {
	store := NewStore(5 * time.Minute)

	success := store.Approve("non-existent", "user", "token")
	if success {
		t.Error("Approve returned true for non-existent session")
	}
}

func TestSessionExpiration(t *testing.T) {
	store := NewStore(100 * time.Millisecond)

	store.Create("expiring-session")

	// Should exist initially
	_, exists := store.Get("expiring-session")
	if !exists {
		t.Error("session should exist initially")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	_, exists = store.Get("expiring-session")
	if exists {
		t.Error("session should be expired")
	}
}

func TestIsExpired(t *testing.T) {
	session := &LoginSession{
		ID:        "test",
		CreatedAt: time.Now(),
	}

	// Not expired yet
	if session.IsExpired(5 * time.Minute) {
		t.Error("session should not be expired")
	}

	// Create expired session
	session.CreatedAt = time.Now().Add(-10 * time.Minute)
	if !session.IsExpired(5 * time.Minute) {
		t.Error("session should be expired")
	}
}

func TestCleanup(t *testing.T) {
	store := NewStore(50 * time.Millisecond)

	store.Create("session1")
	store.Create("session2")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	store.cleanup()

	// Both sessions should be removed
	if len(store.sessions) != 0 {
		t.Errorf("expected 0 sessions after cleanup, got %d", len(store.sessions))
	}
}
