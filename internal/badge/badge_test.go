package badge

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/timfewi/grepquest/internal/game"
	"github.com/timfewi/grepquest/internal/proof"
)

func TestRender(t *testing.T) {
	p := game.NewProfile()
	p.Level = 3
	p.XP = 700
	p.Solved["a"] = true
	p.Loadout[game.Relic] = "codex-ere"
	dir := t.TempDir()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := proof.NewReceipt(p, privateKey.Public().(ed25519.PublicKey), proof.Metadata{SeasonID: "test", PackSHA256: strings.Repeat("a", 64), RulesSHA256: strings.Repeat("b", 64), ClientBuildSHA256: strings.Repeat("c", 64), CreatedAt: time.Unix(0, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if err = receipt.Sign(privateKey); err != nil {
		t.Fatal(err)
	}
	receipt, err = Render(dir, p, receipt)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.ReceiptSHA256) != 64 {
		t.Fatal("invalid hash")
	}
	svg, err := os.ReadFile(filepath.Join(dir, "grepquest.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svg), "LV 3") || !strings.Contains(string(svg), "RELIC · [E] ERE Codex") {
		t.Fatal("level or featured loadout absent from svg")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "grepquest-proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded proof.Receipt
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
}
