package terminal

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// Manager manages multiple terminal sessions (local and SSH).
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewManager creates a terminal session manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// Create starts a new local terminal session and returns its ID.
func (m *Manager) Create() (string, error) {
	id := uuid.New().String()[:8]
	sess, err := newLocalSession(id)
	if err != nil {
		return "", fmt.Errorf("create terminal: %w", err)
	}
	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()
	return id, nil
}

// CreateSSH starts a new SSH terminal session and returns its ID.
func (m *Manager) CreateSSH(cfg SSHConfig) (string, error) {
	id := uuid.New().String()[:8]
	sess, err := newSSHSession(id, cfg)
	if err != nil {
		return "", fmt.Errorf("create ssh terminal: %w", err)
	}
	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()
	return id, nil
}

// SSHConfig holds SSH connection parameters.
type SSHConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	KeyPath  string `json:"keyPath"`
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	sess, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("terminal %s not found", id)
	}
	return sess, nil
}

// Close terminates and removes a session.
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return sess.Close()
}

// List returns all session IDs.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

// CloseAll terminates all sessions.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, sess := range m.sessions {
		sess.Close()
		delete(m.sessions, id)
	}
}
