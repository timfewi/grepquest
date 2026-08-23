package firecracker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
)

// Run with GREPQUEST_FIRECRACKER_INTEGRATION=1 and artifact paths to exercise
// the real KVM boundary. It is skipped in ordinary unit-test environments.
func TestIntegrationRealVM(t *testing.T) {
	if os.Getenv("GREPQUEST_FIRECRACKER_INTEGRATION") != "1" {
		t.Skip("real KVM integration disabled")
	}
	binary := os.Getenv("GREPQUEST_FIRECRACKER")
	kernel := os.Getenv("GREPQUEST_KERNEL")
	initrd := os.Getenv("GREPQUEST_INITRD")
	rootfs := os.Getenv("GREPQUEST_ROOTFS")
	b := New(Config{Binary: binary, Kernel: kernel, Initrd: initrd, RootFS: rootfs, KernelSHA256: fileHash(t, kernel), InitrdSHA256: fileHash(t, initrd), RootFSSHA256: fileHash(t, rootfs), BootTimeout: 15 * time.Second, RunTimeout: 3 * time.Second, LogWriter: os.Stderr})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := challenge.Core()[0]
	if err := b.Boot(ctx, c); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close(context.Background()) }()
	result, err := b.Run(ctx, "grep ERROR vault.log")
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != c.WantOutput || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}
func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
