package main

import (
	"strings"
	"testing"
)

func TestSanitizeErrorRemovesTerminalControls(t *testing.T) {
	got := sanitizeError(fakeError("bad\x1b[31m\x07value"))
	if strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("control characters survived: %q", got)
	}
}
func TestHasParent(t *testing.T) {
	if !hasParent("a/../b") {
		t.Fatal("parent segment accepted")
	}
	if hasParent("a..b/file") {
		t.Fatal("safe filename rejected")
	}
}

type fakeError string

func (e fakeError) Error() string { return string(e) }
