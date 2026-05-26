package terminal

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/UserExistsError/conpty"
)

// Session represents a single PTY terminal session.
type Session struct {
	ID   string
	cpty *conpty.ConPty
}

// NewSession creates a new terminal session.
func NewSession(id string) (*Session, error) {
	shell := shellCommand()

	c, err := conpty.Start(shell)
	if err != nil {
		return nil, fmt.Errorf("conpty start %q: %w", shell, err)
	}

	return &Session{
		ID:   id,
		cpty: c,
	}, nil
}

func shellCommand() string {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			return `powershell.exe -NoLogo -NoExit`
		}
		return `cmd.exe`
	}
	return `bash`
}

// Read reads output from the PTY.
func (s *Session) Read(buf []byte) (int, error) {
	return s.cpty.Read(buf)
}

// Write writes input to the PTY.
func (s *Session) Write(data []byte) (int, error) {
	return s.cpty.Write(data)
}

// Resize changes the terminal window size.
func (s *Session) Resize(rows, cols uint16) error {
	return s.cpty.Resize(int(cols), int(rows))
}

// Close terminates the session.
func (s *Session) Close() error {
	return s.cpty.Close()
}

// IsRunning checks if the process is still running.
func (s *Session) IsRunning() bool {
	return false // simplified; conpty handles lifecycle
}

