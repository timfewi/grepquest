// Package buildinfo exposes immutable build metadata embedded from the source tree.
package buildinfo

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var rawVersion string

// Version returns the canonical source version without trailing whitespace.
func Version() string {
	return strings.TrimSpace(rawVersion)
}
