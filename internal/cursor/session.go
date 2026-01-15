// Package cursor provides Cursor CLI configuration management.
package cursor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Session represents a Cursor CLI session.
type Session struct {
	// ID is the unique chat/session identifier from cursor-agent.
	ID string `json:"id"`

	// WorkDir is the workspace directory where the session was created.
	WorkDir string `json:"work_dir"`

	// Role is the Gas Town role (mayor, polecat, witness, etc.).
	Role string `json:"role,omitempty"`

	// RigName is the rig this session belongs to (empty for town-level sessions).
	RigName string `json:"rig_name,omitempty"`

	// Model is the model used for this session.
	Model string `json:"model,omitempty"`

	// CreatedAt is when the session was created.
	CreatedAt time.Time `json:"created_at"`

	// LastActiveAt is the last time the session was active.
	LastActiveAt time.Time `json:"last_active_at"`

	// Status is the current session status (active, suspended, completed).
	Status string `json:"status"`
}

// SessionStatus constants.
const (
	SessionStatusActive    = "active"
	SessionStatusSuspended = "suspended"
	SessionStatusCompleted = "completed"
)

// SessionStore manages session state persistence.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	path     string
}

// sessionsFileName is the filename for session storage.
const sessionsFileName = "cursor-sessions.json"

// NewSessionStore creates a new session store.
// The store is backed by a JSON file in the given directory.
func NewSessionStore(dir string) (*SessionStore, error) {
	path := filepath.Join(dir, sessionsFileName)
	store := &SessionStore{
		sessions: make(map[string]*Session),
		path:     path,
	}

	// Load existing sessions if file exists
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading sessions: %w", err)
	}

	return store, nil
}

// load reads sessions from disk.
func (s *SessionStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var sessions map[string]*Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return fmt.Errorf("parsing sessions file: %w", err)
	}

	s.mu.Lock()
	s.sessions = sessions
	s.mu.Unlock()

	return nil
}

// save writes sessions to disk.
func (s *SessionStore) save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.sessions, "", "  ")
	s.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshaling sessions: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("creating sessions directory: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("writing sessions file: %w", err)
	}

	return nil
}

// Get returns a session by ID.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

// GetByRole returns the most recent active session for a role.
func (s *SessionStore) GetByRole(role, rigName string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best *Session
	for _, sess := range s.sessions {
		if sess.Role != role || sess.RigName != rigName {
			continue
		}
		if sess.Status != SessionStatusActive {
			continue
		}
		if best == nil || sess.LastActiveAt.After(best.LastActiveAt) {
			best = sess
		}
	}
	return best
}

// Put stores a session.
func (s *SessionStore) Put(sess *Session) error {
	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()
	return s.save()
}

// Delete removes a session.
func (s *SessionStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	return s.save()
}

// List returns all sessions.
func (s *SessionStore) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		result = append(result, sess)
	}
	return result
}

// ListActive returns all active sessions.
func (s *SessionStore) ListActive() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Session, 0)
	for _, sess := range s.sessions {
		if sess.Status == SessionStatusActive {
			result = append(result, sess)
		}
	}
	return result
}

// CleanupStale removes sessions older than the given duration.
func (s *SessionStore) CleanupStale(maxAge time.Duration) error {
	s.mu.Lock()
	cutoff := time.Now().Add(-maxAge)
	for id, sess := range s.sessions {
		if sess.LastActiveAt.Before(cutoff) {
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()
	return s.save()
}

// CaptureSessionID attempts to capture the session ID from cursor-agent output.
// This is called from the stop hook to record the session for potential resume.
//
// cursor-agent outputs session info in various ways:
// - In JSON output mode: {"chat_id": "..."}
// - In text mode: Look for patterns like "Session: abc123" or "Chat ID: abc123"
func CaptureSessionID(output string) string {
	// Try JSON parsing first
	var data struct {
		ChatID string `json:"chat_id"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal([]byte(output), &data); err == nil {
		if data.ChatID != "" {
			return data.ChatID
		}
		if data.ID != "" {
			return data.ID
		}
	}

	// Look for common patterns in text output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Pattern: "Session: abc123" or "Chat ID: abc123"
		for _, prefix := range []string{"Session:", "Chat ID:", "ChatID:", "session:", "chat_id:"} {
			if strings.HasPrefix(line, prefix) {
				id := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				if id != "" {
					return id
				}
			}
		}

		// Pattern: "Resuming session abc123"
		if strings.Contains(line, "session") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "session" && i+1 < len(parts) {
					return parts[i+1]
				}
			}
		}
	}

	return ""
}

// ListCursorSessions runs 'cursor-agent ls' to list available sessions.
// Note: This may not work in non-TTY environments.
func ListCursorSessions() ([]string, error) {
	cmd := exec.Command("cursor-agent", "ls")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing cursor sessions: %w", err)
	}

	var sessions []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "ID") { // Skip header
			// Extract first field (session ID)
			fields := strings.Fields(line)
			if len(fields) > 0 {
				sessions = append(sessions, fields[0])
			}
		}
	}

	return sessions, nil
}

// ResumeSession builds a command to resume a cursor-agent session.
func ResumeSession(sessionID string, args ...string) []string {
	cmdArgs := []string{"--resume", sessionID}
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}

// CreateChat creates a new cursor-agent chat and returns its ID.
// This uses 'cursor-agent create-chat' if available.
func CreateChat(workDir string) (string, error) {
	cmd := exec.Command("cursor-agent", "create-chat")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("creating chat: %w", err)
	}

	// Parse output for chat ID
	id := CaptureSessionID(string(output))
	if id == "" {
		// Try to extract from raw output
		id = strings.TrimSpace(string(output))
	}

	if id == "" {
		return "", fmt.Errorf("could not extract chat ID from output")
	}

	return id, nil
}

// GetLatestSession returns the most recent cursor-agent session.
func GetLatestSession() (string, error) {
	sessions, err := ListCursorSessions()
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions found")
	}
	return sessions[0], nil // First is typically most recent
}

// SessionFromEnv creates a Session from environment variables.
// This is used during session startup to capture context.
func SessionFromEnv(workDir, role, rigName string) *Session {
	return &Session{
		WorkDir:      workDir,
		Role:         role,
		RigName:      rigName,
		Model:        os.Getenv("CURSOR_MODEL"),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
		Status:       SessionStatusActive,
	}
}

// Touch updates the LastActiveAt timestamp.
func (s *Session) Touch() {
	s.LastActiveAt = time.Now()
}

// MarkCompleted marks the session as completed.
func (s *Session) MarkCompleted() {
	s.Status = SessionStatusCompleted
	s.LastActiveAt = time.Now()
}

// MarkSuspended marks the session as suspended.
func (s *Session) MarkSuspended() {
	s.Status = SessionStatusSuspended
	s.LastActiveAt = time.Now()
}
