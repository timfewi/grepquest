package native

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/commandparse"
	"github.com/timfewi/grepquest/internal/sandbox"
)

const outputLimit = 64 << 10

type Backend struct {
	root     string
	current  challenge.Challenge
	grepPath string
}

func New() *Backend                          { path, _ := exec.LookPath("grep"); return &Backend{grepPath: path} }
func NewWithExecutable(path string) *Backend { return &Backend{grepPath: path} }

func (b *Backend) Boot(_ context.Context, c challenge.Challenge) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := b.Close(context.Background()); err != nil {
		return err
	}
	root, err := os.MkdirTemp("", "grepquest-native-")
	if err != nil {
		return err
	}
	b.root, b.current = root, c
	for name, contents := range c.Fixtures {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			_ = b.Close(context.Background())
			return err
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			_ = b.Close(context.Background())
			return err
		}
	}
	return nil
}

func (b *Backend) Run(ctx context.Context, command string) (sandbox.ExecutionResult, error) {
	if b.root == "" {
		return sandbox.ExecutionResult{}, errors.New("sandbox is not booted")
	}
	if b.grepPath == "" {
		return sandbox.ExecutionResult{}, errors.New("grep executable was not found in PATH")
	}
	argv, err := commandparse.Split(command)
	if err != nil {
		return sandbox.ExecutionResult{}, err
	}
	if len(argv) < 2 || argv[0] != "grep" {
		return sandbox.ExecutionResult{}, errors.New("only a single grep command is allowed")
	}
	for _, arg := range argv[1:] {
		if strings.ContainsRune(arg, '\x00') {
			return sandbox.ExecutionResult{}, errors.New("NUL bytes are not allowed")
		}
		if filepath.IsAbs(arg) || containsParent(arg) {
			return sandbox.ExecutionResult{}, fmt.Errorf("path escape is not allowed: %q", arg)
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, b.grepPath, argv[1:]...)
	cmd.Dir = b.root
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C", "LANG=C", "HOME=" + b.root}
	stdout, stderr := &limitBuffer{limit: outputLimit}, &limitBuffer{limit: outputLimit}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	started := time.Now()
	runErr := cmd.Run()
	duration := time.Since(started)
	result := sandbox.ExecutionResult{Stdout: stdout.String(), Stderr: stderr.String(), Args: append([]string(nil), argv...), ExitCode: 0, Duration: duration, Truncated: stdout.truncated || stderr.truncated}
	if runCtx.Err() == context.DeadlineExceeded {
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
		return result, errors.New("command output exceeded 64 KiB")
	}
	return result, nil
}

func (b *Backend) Reset(ctx context.Context) error {
	if b.root == "" {
		return errors.New("sandbox is not booted")
	}
	return b.Boot(ctx, b.current)
}
func (b *Backend) Close(_ context.Context) error {
	if b.root == "" {
		return nil
	}
	root := b.root
	b.root = ""
	return os.RemoveAll(root)
}

func containsParent(s string) bool {
	for _, p := range strings.Split(filepath.ToSlash(s), "/") {
		if p == ".." {
			return true
		}
	}
	return false
}

type limitBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.truncated = true
		return original, nil
	}
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, err := b.Buffer.Write(p)
	return original, err
}

var _ io.Writer = (*limitBuffer)(nil)
