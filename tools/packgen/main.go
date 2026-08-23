// Command packgen materializes the compiled-in authored core pack. It is kept
// as a generator so the canonical JSON reviewed and shipped in releases cannot
// silently diverge from the offline tutorial used by the application.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/timfewi/grepquest/internal/challenge"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	root := filepath.Join("packs", "core", "v1")
	dir := filepath.Join(root, "challenges")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	manifest, err := challenge.CoreManifest("GNU grep 3.12")
	if err != nil {
		return err
	}
	for _, c := range challenge.Core() {
		b, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		if err = os.WriteFile(filepath.Join(dir, c.ID+".json"), b, 0o644); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "pack.json"), append(b, '\n'), 0o644)
}
