package game

type Slot string

const (
	Head  Slot = "head"
	Tool  Slot = "tool"
	Body  Slot = "body"
	Charm Slot = "charm"
	Relic Slot = "relic"
)

type Item struct {
	ID, Name, Sprite, Flavor, Skill string
	Slot                            Slot
	Rarity                          string
}

var BuiltinItems = []Item{
	{"lens-brass", "Brass Pattern Lens", "[o]", "Separates signal from cellar dust.", "literal", Head, "common"},
	{"lens-ere", "Foundry Monocle", "[@]", "Alternatives shimmer at the edge.", "ere", Head, "rare"},
	{"compass-root", "Rootbound Compass", "-+>", "Its needle points through directory trees.", "recursive", Tool, "common"},
	{"compass-fixed", "Literal Chisel", "==>", "No metacharacter escapes its edge.", "fixed", Tool, "uncommon"},
	{"mantle-context", "Mantle of Before & After", "/##\\", "Keeps nearby lines within reach.", "context", Body, "rare"},
	{"mantle-signal", "Signal Jerkin", "/++\\", "Woven from correctly selected lines.", "literal", Body, "common"},
	{"sigil-zero", "Zero Sigil", "(0)", "Glows when the search succeeds.", "exit-code", Charm, "common"},
	{"sigil-one", "One Sigil", "(1)", "Honors an honest absence.", "exit-code", Charm, "uncommon"},
	{"codex-bre", "BRE Codex", "[B]", "Old runes, precisely escaped.", "bre", Relic, "rare"},
	{"codex-ere", "ERE Codex", "[E]", "Forked paths bound in green leather.", "ere", Relic, "epic"},
}

func AwardFor(challengeID string, clearCount int) Item {
	return BuiltinItems[(stableIndex(challengeID)+clearCount-1)%len(BuiltinItems)]
}
func stableIndex(s string) int {
	n := 0
	for _, r := range s {
		n = (n*31 + int(r)) & 0x7fffffff
	}
	return n
}
