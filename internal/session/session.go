package session

import (
	"sync"
	"time"
)

// Status represents a login session status
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusExpired  Status = "expired"
)

// LoginSession represents a login session
type LoginSession struct {
	ID          string
	Status      Status
	Username    string
	AccessToken string
	CreatedAt   time.Time
}

// IsExpired checks if the session has expired
func (s *LoginSession) IsExpired(maxAge time.Duration) bool {
	return time.Since(s.CreatedAt) > maxAge
}

// Store manages login sessions in memory
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*LoginSession
	maxAge   time.Duration
}

// NewStore creates a new session store
func NewStore(maxAge time.Duration) *Store {
	if maxAge == 0 {
		maxAge = 15 * time.Minute
	}
	s := &Store{
		sessions: make(map[string]*LoginSession),
		maxAge:   maxAge,
	}
	go s.cleanupLoop()
	return s
}

// Create creates a new login session
func (s *Store) Create(id string) *LoginSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &LoginSession{
		ID:        id,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
	s.sessions[id] = session
	return session
}

// Get retrieves a login session
func (s *Store) Get(id string) (*LoginSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, false
	}

	if session.IsExpired(s.maxAge) {
		return nil, false
	}

	return session, true
}

// Approve marks a session as approved
func (s *Store) Approve(id, username, accessToken string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	if !exists {
		return false
	}

	if session.IsExpired(s.maxAge) {
		return false
	}

	session.Status = StatusApproved
	session.Username = username
	session.AccessToken = accessToken
	return true
}

// cleanupLoop periodically removes expired sessions
func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes expired sessions
func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		if session.IsExpired(s.maxAge) {
			delete(s.sessions, id)
		}
	}
}
