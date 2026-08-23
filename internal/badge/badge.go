package badge

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/timfewi/grepquest/internal/game"
	"github.com/timfewi/grepquest/internal/proof"
)

func Render(dir string, p game.Profile, receipt proof.Receipt) (proof.Receipt, error) {
	if err := receipt.Verify(); err != nil {
		return proof.Receipt{}, err
	}
	proofJSON, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return proof.Receipt{}, err
	}
	proofJSON = append(proofJSON, '\n')
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return proof.Receipt{}, err
	}
	loadout := html.EscapeString(featuredLoadout(p))
	tier := html.EscapeString(strings.ToUpper(receipt.ProofTier))
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="640" height="120" role="img" aria-label="GrepQuest level %d">
<rect width="640" height="120" rx="10" fill="#10151c"/>
<text x="22" y="32" fill="#65d46e" font-family="monospace" font-size="20">GREPQUEST · LV %d</text>
<text x="22" y="60" fill="#d8dee9" font-family="monospace" font-size="14">%d rooms cleared · %d XP</text>
<text x="22" y="84" fill="#9aa7b5" font-family="monospace" font-size="12">%s</text>
<text x="22" y="105" fill="#7f8c9d" font-family="monospace" font-size="12">%s · #%s</text>
<text x="535" y="39" fill="#65d46e" font-family="monospace" font-size="18"> /\</text>
<text x="535" y="62" fill="#65d46e" font-family="monospace" font-size="18">/[]\</text>
<text x="535" y="85" fill="#65d46e" font-family="monospace" font-size="18"> /\</text>
<text x="535" y="105" fill="#65d46e" font-family="monospace" font-size="18">/  \</text>
</svg>
`, p.Level, p.Level, solvedCount(p), p.XP, loadout, tier, html.EscapeString(receipt.ReceiptSHA256[:8]))
	if err = os.WriteFile(filepath.Join(dir, "grepquest-proof.json"), proofJSON, 0o644); err != nil {
		return proof.Receipt{}, err
	}
	if err = os.WriteFile(filepath.Join(dir, "grepquest.svg"), []byte(svg), 0o644); err != nil {
		return proof.Receipt{}, err
	}
	return receipt, nil
}

func featuredLoadout(p game.Profile) string {
	for _, slot := range []game.Slot{game.Relic, game.Tool, game.Head, game.Body, game.Charm} {
		id := p.Loadout[slot]
		for _, item := range game.BuiltinItems {
			if item.ID == id {
				return strings.ToUpper(string(slot)) + " · " + item.Sprite + " " + item.Name
			}
		}
	}
	return "LOADOUT · UNGEARED"
}

func solvedCount(p game.Profile) int {
	count := 0
	for _, solved := range p.Solved {
		if solved {
			count++
		}
	}
	return count
}
