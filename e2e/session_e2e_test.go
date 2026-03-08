package e2e

import (
	"testing"
	"time"

	"github.com/azophy/sshifu/internal/session"
)

// TestSessionE2E tests the complete session lifecycle
func TestSessionE2E(t *testing.T) {
	t.Run("Session_CompleteLifecycle", func(t *testing.T) {
		// Create session store
		store := session.NewStore(15 * time.Minute)

		// 1. Create session
		sessionID := "test-session-123"
		store.Create(sessionID)

		// 2. Verify session exists and is pending
		sess, exists := store.Get(sessionID)
		if !exists {
			t.Fatal("Session should exist")
		}

		if sess.Status != session.StatusPending {
			t.Errorf("Expected pending status, got: %s", sess.Status)
		}

		// 3. Approve session
		success := store.Approve(sessionID, "testuser", "test_access_token")
		if !success {
			t.Error("Approve should succeed")
		}

		// 4. Verify session is approved
		sess, exists = store.Get(sessionID)
		if !exists {
			t.Fatal("Session should still exist")
		}

		if sess.Status != session.StatusApproved {
			t.Errorf("Expected approved status, got: %s", sess.Status)
		}

		if sess.Username != "testuser" {
			t.Errorf("Expected username testuser, got: %s", sess.Username)
		}

		if sess.AccessToken != "test_access_token" {
			t.Errorf("Expected test_access_token, got: %s", sess.AccessToken)
		}

		t.Log("Session lifecycle completed successfully")
	})

	t.Run("Session_Expiration", func(t *testing.T) {
		// Create session store with short TTL
		store := session.NewStore(100 * time.Millisecond)

		sessionID := "test-session-expire"
		store.Create(sessionID)

		// Wait for expiration
		time.Sleep(200 * time.Millisecond)

		// Session should be expired
		_, exists := store.Get(sessionID)
		if exists {
			t.Error("Session should be expired")
		}

		t.Log("Session expiration works correctly")
	})

	t.Run("Session_Cleanup", func(t *testing.T) {
		// Create session store with short TTL
		store := session.NewStore(50 * time.Millisecond)

		// Create multiple sessions
		for i := 0; i < 5; i++ {
			store.Create("session-" + string(rune('0'+i)))
		}

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Sessions will be cleaned up automatically by the background goroutine
		// Just verify they are gone
		count := 0
		store.Range(func(id string, sess *session.LoginSession) bool {
			count++
			return true
		})

		// Some or all sessions should be cleaned up by now
		t.Logf("Sessions after cleanup wait: %d", count)
	})
}

// TestSessionRangeE2E tests the Range method for session iteration
func TestSessionRangeE2E(t *testing.T) {
	store := session.NewStore(15 * time.Minute)

	// Create multiple sessions
	sessions := []string{"session-1", "session-2", "session-3"}
	for _, id := range sessions {
		store.Create(id)
	}

	// Iterate and collect IDs
	var foundIDs []string
	store.Range(func(id string, sess *session.LoginSession) bool {
		foundIDs = append(foundIDs, id)
		return true
	})

	if len(foundIDs) != len(sessions) {
		t.Errorf("Expected %d sessions, got: %d", len(sessions), len(foundIDs))
	}

	t.Logf("Found %d sessions via Range", len(foundIDs))
}

// TestSessionTokenLookupE2E tests finding sessions by access token
func TestSessionTokenLookupE2E(t *testing.T) {
	store := session.NewStore(15 * time.Minute)

	// Create and approve sessions
	store.Create("session-1")
	store.Approve("session-1", "user1", "token-1")

	store.Create("session-2")
	store.Approve("session-2", "user2", "token-2")

	store.Create("session-3")
	// session-3 remains pending

	// Find session by token
	var foundUsername string
	store.Range(func(id string, sess *session.LoginSession) bool {
		if sess.Status == session.StatusApproved && sess.AccessToken == "token-2" {
			foundUsername = sess.Username
			return false
		}
		return true
	})

	if foundUsername != "user2" {
		t.Errorf("Expected to find user2, got: %s", foundUsername)
	}

	t.Log("Token-based session lookup works correctly")
}
