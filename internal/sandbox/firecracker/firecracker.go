// Package firecracker implements the production Linux/KVM sandbox boundary.
// The guest image must run the GrepQuest agent on vsock port 52. No network
// device or host filesystem share is configured by this backend.
package firecracker

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/sandbox"
)

const guestPort = 52

type Config struct {
	Binary, Kernel, Initrd, RootFS           string
	KernelSHA256, InitrdSHA256, RootFSSHA256 string
	VCPU, MemoryMiB                          int
	BootTimeout, RunTimeout                  time.Duration
	LogWriter                                io.Writer
}

type Backend struct {
	config                      Config
	dir, apiSocket, vsockSocket string
	cmd                         *exec.Cmd
	current                     challenge.Challenge
}

func New(config Config) *Backend {
	if config.VCPU == 0 {
		config.VCPU = 1
	}
	if config.MemoryMiB == 0 {
		config.MemoryMiB = 256
	}
	if config.BootTimeout == 0 {
		config.BootTimeout = 5 * time.Second
	}
	if config.RunTimeout == 0 {
		config.RunTimeout = 2 * time.Second
	}
	return &Backend{config: config}
}

func (b *Backend) Preflight() error {
	if runtimeGOOS() != "linux" {
		return errors.New("firecracker requires Linux")
	}
	if b.config.Binary == "" || b.config.Kernel == "" || b.config.Initrd == "" || b.config.RootFS == "" {
		return errors.New("firecracker binary, kernel, initrd, and rootfs paths are required")
	}
	for _, item := range []struct{ label, path string }{{"binary", b.config.Binary}, {"kernel", b.config.Kernel}, {"initrd", b.config.Initrd}, {"rootfs", b.config.RootFS}} {
		label, path := item.label, item.path
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%s unavailable: %w", label, err)
		}
		if label == "binary" && info.Mode()&0o111 == 0 {
			return errors.New("firecracker binary is not executable")
		}
	}
	kvm, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("KVM unavailable: %w", err)
	}
	_ = kvm.Close()
	if err := verifyDigest(b.config.Kernel, b.config.KernelSHA256); err != nil {
		return fmt.Errorf("kernel: %w", err)
	}
	if err := verifyDigest(b.config.Initrd, b.config.InitrdSHA256); err != nil {
		return fmt.Errorf("initrd: %w", err)
	}
	if err := verifyDigest(b.config.RootFS, b.config.RootFSSHA256); err != nil {
		return fmt.Errorf("rootfs: %w", err)
	}
	return nil
}

func (b *Backend) Boot(ctx context.Context, c challenge.Challenge) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := b.Preflight(); err != nil {
		return err
	}
	if err := b.Close(context.Background()); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "grepquest-firecracker-")
	if err != nil {
		return err
	}
	b.dir = dir
	b.apiSocket = filepath.Join(dir, "api.sock")
	b.vsockSocket = filepath.Join(dir, "vsock.sock")
	b.current = c
	b.cmd = exec.CommandContext(ctx, b.config.Binary, "--api-sock", b.apiSocket)
	b.cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C"}
	logWriter := b.config.LogWriter
	if logWriter == nil {
		logWriter = io.Discard
	}
	b.cmd.Stdout = logWriter
	b.cmd.Stderr = logWriter
	if err = b.cmd.Start(); err != nil {
		_ = b.Close(context.Background())
		return fmt.Errorf("start firecracker: %w", err)
	}
	bootCtx, cancel := context.WithTimeout(ctx, b.config.BootTimeout)
	defer cancel()
	if err = waitForSocket(bootCtx, b.apiSocket); err != nil {
		_ = b.Close(context.Background())
		return err
	}
	client := unixHTTPClient(b.apiSocket)
	requests := []struct {
		path string
		body any
	}{
		{"/machine-config", map[string]any{"vcpu_count": b.config.VCPU, "mem_size_mib": b.config.MemoryMiB, "smt": false, "track_dirty_pages": false}},
		{"/boot-source", map[string]any{"kernel_image_path": b.config.Kernel, "initrd_path": b.config.Initrd, "boot_args": "console=ttyS0 reboot=k panic=1 root=/dev/vda ro init=/sbin/init random.trust_cpu=on"}},
		{"/drives/rootfs", map[string]any{"drive_id": "rootfs", "path_on_host": b.config.RootFS, "is_root_device": true, "is_read_only": true}},
		{"/vsock", map[string]any{"guest_cid": 3, "uds_path": b.vsockSocket}},
		{"/actions", map[string]any{"action_type": "InstanceStart"}},
	}
	for _, req := range requests {
		if err = putJSON(bootCtx, client, req.path, req.body); err != nil {
			_ = b.Close(context.Background())
			return fmt.Errorf("configure %s: %w", req.path, err)
		}
	}
	if err = waitForSocket(bootCtx, b.vsockSocket); err != nil {
		_ = b.Close(context.Background())
		return fmt.Errorf("guest vsock: %w", err)
	}
	if err = b.waitForAgent(bootCtx); err != nil {
		_ = b.Close(context.Background())
		return fmt.Errorf("guest agent readiness: %w", err)
	}
	guestChallenge := c
	guestChallenge.HiddenTests = nil
	if err = b.callAgent(bootCtx, agentRequest{Type: "boot", Challenge: &guestChallenge}, nil); err != nil {
		_ = b.Close(context.Background())
		return fmt.Errorf("boot guest agent: %w", err)
	}
	return nil
}

func (b *Backend) waitForAgent(ctx context.Context) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		if err := b.callAgent(ctx, agentRequest{Type: "ready"}, nil); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last connection error: %v)", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func (b *Backend) Run(ctx context.Context, command string) (sandbox.ExecutionResult, error) {
	if b.cmd == nil {
		return sandbox.ExecutionResult{}, errors.New("sandbox is not booted")
	}
	runCtx, cancel := context.WithTimeout(ctx, b.config.RunTimeout)
	defer cancel()
	var result sandbox.ExecutionResult
	err := b.callAgent(runCtx, agentRequest{Type: "run", Command: command}, &result)
	if runCtx.Err() == context.DeadlineExceeded {
		return result, errors.New("guest command timed out")
	}
	return result, err
}
func (b *Backend) Reset(ctx context.Context) error {
	if b.cmd == nil {
		return errors.New("sandbox is not booted")
	}
	c := b.current
	return b.Boot(ctx, c)
}
func (b *Backend) Close(_ context.Context) error {
	if b.cmd != nil && b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
		_, _ = b.cmd.Process.Wait()
	}
	b.cmd = nil
	if b.dir != "" {
		removeErr := os.RemoveAll(b.dir)
		if removeErr != nil {
			return removeErr
		}
	}
	b.dir = ""
	return nil
}

type agentRequest struct {
	Type      string               `json:"type"`
	Command   string               `json:"command,omitempty"`
	Challenge *challenge.Challenge `json:"challenge,omitempty"`
}

func (b *Backend) callAgent(ctx context.Context, request agentRequest, response any) error {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", b.vsockSocket)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	if _, err = fmt.Fprintf(conn, "CONNECT %d\n", guestPort); err != nil {
		return err
	}
	reader := bufio.NewReader(io.LimitReader(conn, 128<<10))
	status, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !validVsockStatus(status) {
		return fmt.Errorf("vsock rejected connection: %q", strings.TrimSpace(status))
	}
	if err = json.NewEncoder(conn).Encode(request); err != nil {
		return err
	}
	if response == nil {
		var ack struct {
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		}
		if err = json.NewDecoder(reader).Decode(&ack); err != nil {
			return err
		}
		if !ack.OK {
			return errors.New(ack.Error)
		}
		return nil
	}
	return json.NewDecoder(reader).Decode(response)
}

func validVsockStatus(status string) bool {
	fields := strings.Fields(status)
	if len(fields) != 2 || fields[0] != "OK" {
		return false
	}
	port, err := strconv.ParseUint(fields[1], 10, 32)
	return err == nil && port != 0
}

func verifyDigest(path, want string) error {
	if len(want) != 64 {
		return errors.New("a full SHA-256 digest is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("SHA-256 mismatch: got %s", got)
	}
	return nil
}
func waitForSocket(ctx context.Context, path string) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeSocket != 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for %s: %w", path, ctx.Err())
		case <-ticker.C:
		}
	}
}
func unixHTTPClient(socket string) *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
}
func putJSON(ctx context.Context, client *http.Client, path string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, "http://localhost"+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}
	return nil
}
