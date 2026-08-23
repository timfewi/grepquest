package proof

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/timfewi/grepquest/internal/game"
)

func TestSignVerifyAndTamper(t *testing.T) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	profile := game.NewProfile()
	profile.XP = 420
	profile.Mastery = map[string]int{"ere": 82, "literal": 91}
	receipt, err := NewReceipt(profile, key.Public().(ed25519.PublicKey), metadata())
	if err != nil {
		t.Fatal(err)
	}
	if err = receipt.Sign(key); err != nil {
		t.Fatal(err)
	}
	if err = receipt.Verify(); err != nil {
		t.Fatal(err)
	}
	receipt.Score++
	if err = receipt.Verify(); err == nil {
		t.Fatal("tampered receipt verified")
	}
}
func TestCanonicalMasteryOrder(t *testing.T) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	a := game.NewProfile()
	a.Mastery = map[string]int{"z": 1, "a": 2}
	b := game.NewProfile()
	b.Mastery = map[string]int{"a": 2, "z": 1}
	ra, _ := NewReceipt(a, key.Public().(ed25519.PublicKey), metadata())
	rb, _ := NewReceipt(b, key.Public().(ed25519.PublicKey), metadata())
	ca, _ := ra.Canonical()
	cb, _ := rb.Canonical()
	if string(ca) != string(cb) {
		t.Fatalf("canonical bytes differ\n%s\n%s", ca, cb)
	}
}
func TestKeyPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key")
	key, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != ed25519.PrivateKeySize {
		t.Fatal("wrong key size")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	if err = os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadOrCreateKey(path); err == nil {
		t.Fatal("unsafe key permissions accepted")
	}
}
func metadata() Metadata {
	return Metadata{SeasonID: "test", PackSHA256: strings.Repeat("a", 64), RulesSHA256: strings.Repeat("b", 64), ClientBuildSHA256: strings.Repeat("c", 64), CreatedAt: time.Unix(0, 0)}
}
