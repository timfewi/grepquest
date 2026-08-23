// GrepQuest guest agent is the trusted judge-side process inside the microVM.
// It exposes only challenge installation and one bounded GNU grep invocation.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/commandparse"
	"github.com/timfewi/grepquest/internal/sandbox"
	"golang.org/x/sys/unix"
)

const (
	port          = 52
	playerUID     = 1000
	playerGID     = 1000
	challengeRoot = "/run/grepquest/challenge"
	maxRequest    = 1 << 20
	maxOutput     = 64 << 10
)

type request struct {
	Type      string               `json:"type"`
	Command   string               `json:"command,omitempty"`
	Challenge *challenge.Challenge `json:"challenge,omitempty"`
}
type ack struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func main() {
	if err := serve(); err != nil {
		fmt.Fprintln(os.Stderr, "grepquest-agent:", err)
		os.Exit(1)
	}
}
func serve() error {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(fd) }()
	if err = unix.Bind(fd, &unix.SockaddrVM{CID: unix.VMADDR_CID_ANY, Port: port}); err != nil {
		return err
	}
	if err = unix.Listen(fd, 8); err != nil {
		return err
	}
	for {
		client, _, err := unix.Accept4(fd, unix.SOCK_CLOEXEC)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return err
		}
		go handle(&fdConn{fd: client})
	}
}

func handle(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var req request
	if err := json.NewDecoder(io.LimitReader(conn, maxRequest)).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(ack{Error: "malformed request"})
		return
	}
	switch req.Type {
	case "ready":
		_ = json.NewEncoder(conn).Encode(ack{OK: true})
	case "boot":
		err := install(req.Challenge)
		if err != nil {
			_ = json.NewEncoder(conn).Encode(ack{Error: err.Error()})
		} else {
			_ = json.NewEncoder(conn).Encode(ack{OK: true})
		}
	case "run":
		result, err := run(req.Command)
		if err != nil {
			result.Stderr = sanitizeError(err)
		}
		_ = json.NewEncoder(conn).Encode(result)
	default:
		_ = json.NewEncoder(conn).Encode(ack{Error: "unknown request type"})
	}
}

func install(c *challenge.Challenge) error {
	if c == nil {
		return errors.New("challenge is required")
	}
	copyChallenge := *c
	copyChallenge.HiddenTests = nil
	if err := copyChallenge.Validate(); err != nil {
		return err
	}
	if err := os.RemoveAll(challengeRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(challengeRoot, 0o500); err != nil {
		return err
	}
	for name, contents := range copyChallenge.Fixtures {
		path := filepath.Join(challengeRoot, filepath.FromSlash(name))
		if !strings.HasPrefix(path, challengeRoot+string(filepath.Separator)) {
			return errors.New("fixture escaped challenge root")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o500); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(contents), 0o400); err != nil {
			return err
		}
	}
	return filepath.Walk(challengeRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(path, playerUID, playerGID)
	})
}

func run(command string) (sandbox.ExecutionResult, error) {
	argv, err := commandparse.Split(command)
	if err != nil {
		return sandbox.ExecutionResult{}, err
	}
	if len(argv) < 2 || argv[0] != "grep" {
		return sandbox.ExecutionResult{}, errors.New("only a single grep command is allowed")
	}
	for _, arg := range argv[1:] {
		if strings.ContainsRune(arg, '\x00') || filepath.IsAbs(arg) || hasParent(arg) {
			return sandbox.ExecutionResult{}, errors.New("path escape is not allowed")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/grep", argv[1:]...)
	cmd.Dir = challengeRoot
	cmd.Env = []string{"PATH=/bin", "LC_ALL=C", "LANG=C", "HOME=/run/grepquest/player"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: playerUID, Gid: playerGID, NoSetGroups: true}, Setpgid: true, Pdeathsig: syscall.SIGKILL}
	stdout, stderr := &limitBuffer{limit: maxOutput}, &limitBuffer{limit: maxOutput}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	started := time.Now()
	runErr := cmd.Run()
	result := sandbox.ExecutionResult{Stdout: stdout.String(), Stderr: stderr.String(), Args: append([]string(nil), argv...), Duration: time.Since(started), Truncated: stdout.truncated || stderr.truncated}
	if ctx.Err() == context.DeadlineExceeded {
		result.ExitCode = 124
		return result, errors.New("command timed out")
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, runErr
		}
	}
	if result.Truncated {
		return result, errors.New("output limit exceeded")
	}
	return result, nil
}

func hasParent(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
func sanitizeError(err error) string {
	s := strings.Map(func(r rune) rune {
		if r < ' ' && r != '\n' && r != '\t' {
			return -1
		}
		if r == 0x7f {
			return -1
		}
		return r
	}, err.Error())
	if len(s) > 512 {
		s = s[:512]
	}
	return s
}

type limitBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return n, nil
	}
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, err := b.Buffer.Write(p)
	return n, err
}

type fdConn struct{ fd int }

func (c *fdConn) Read(p []byte) (int, error) {
	n, err := unix.Read(c.fd, p)
	if n < 0 {
		n = 0
	}
	if err == nil && n == 0 {
		return 0, io.EOF
	}
	return n, err
}
func (c *fdConn) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		n, err := unix.Write(c.fd, p)
		if n > 0 {
			written += n
			p = p[n:]
		}
		if err != nil {
			return written, err
		}
		if n == 0 {
			return written, io.ErrUnexpectedEOF
		}
	}
	return written, nil
}
func (c *fdConn) Close() error { return unix.Close(c.fd) }
