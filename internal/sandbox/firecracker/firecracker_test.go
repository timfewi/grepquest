package firecracker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreflightFailsClosedWithoutConfig(t *testing.T) {
	if err := New(Config{}).Preflight(); err == nil {
		t.Fatal("missing artifacts accepted")
	}
}
func TestDigestRequiredAndVerified(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image")
	if err := os.WriteFile(path, []byte("guest"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyDigest(path, strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := verifyDigest(path, "84983c60f7daadc1cb8698621f802c0d9f9a3c3c295c810748fb048115c186ec"); err != nil {
		t.Fatal(err)
	}
}
