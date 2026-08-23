package challenge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

type Manifest struct {
	Version, ID, Title, GrepVersion, License, PackSHA256 string
	Challenges                                           []ManifestChallenge
}
type ManifestChallenge struct{ ID, SHA256 string }

func CoreManifest(grepVersion string) (Manifest, error) {
	items := Core()
	entries := make([]ManifestChallenge, 0, len(items))
	for _, c := range items {
		hash, err := c.Hash()
		if err != nil {
			return Manifest{}, err
		}
		entries = append(entries, ManifestChallenge{c.ID, hash})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	canonical, err := json.Marshal(struct {
		Version, ID, GrepVersion string
		Challenges               []ManifestChallenge
	}{"1", "core", grepVersion, entries})
	if err != nil {
		return Manifest{}, err
	}
	sum := sha256.Sum256(canonical)
	return Manifest{Version: "1", ID: "core", Title: "The Signal Cellar & Regex Foundry", GrepVersion: grepVersion, License: "CC-BY-4.0", PackSHA256: hex.EncodeToString(sum[:]), Challenges: entries}, nil
}

func VerifyManifest(m Manifest, challenges []Challenge) error {
	if len(m.Challenges) != len(challenges) {
		return fmt.Errorf("manifest lists %d challenges, loaded %d", len(m.Challenges), len(challenges))
	}
	byID := map[string]Challenge{}
	for _, c := range challenges {
		byID[c.ID] = c
	}
	for _, entry := range m.Challenges {
		c, ok := byID[entry.ID]
		if !ok {
			return fmt.Errorf("manifest challenge %q missing", entry.ID)
		}
		hash, err := c.Hash()
		if err != nil {
			return err
		}
		if hash != entry.SHA256 {
			return fmt.Errorf("challenge %q hash mismatch", entry.ID)
		}
	}
	return nil
}
