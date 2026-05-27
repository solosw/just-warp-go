package terminal

import (
	"os/exec"
	"runtime"

	"github.com/UserExistsError/conpty"
)

// terminalIO is the internal interface for Read/Write/Resize/Close operations.
type terminalIO interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Resize(cols, rows uint16) error
	Close() error
}

// Session represents a single terminal session (local or SSH).
type Session struct {
	ID string
	io terminalIO
}

// newLocalSession creates a local ConPTY session.
func newLocalSession(id string) (*Session, error) {
	shell := "bash"
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			shell = `powershell.exe -NoLogo -NoExit`
		} else {
			shell = `cmd.exe`
		}
	}
	c, err := conpty.Start(shell)
	if err != nil {
		return nil, err
	}
	return &Session{ID: id, io: &localIO{cpty: c}}, nil
}

type localIO struct {
	cpty *conpty.ConPty
}

func (l *localIO) Read(buf []byte) (int, error)  { return l.cpty.Read(buf) }
func (l *localIO) Write(data []byte) (int, error) { return l.cpty.Write(data) }
func (l *localIO) Resize(cols, rows uint16) error { return l.cpty.Resize(int(cols), int(rows)) }
func (l *localIO) Close() error                   { return l.cpty.Close() }

// Read reads output from the session.
func (s *Session) Read(buf []byte) (int, error) { return s.io.Read(buf) }

// Write writes input to the session.
func (s *Session) Write(data []byte) (int, error) { return s.io.Write(data) }

// Resize changes the terminal window size.
func (s *Session) Resize(rows, cols uint16) error { return s.io.Resize(cols, rows) }

// Close terminates the session.
func (s *Session) Close() error { return s.io.Close() }
