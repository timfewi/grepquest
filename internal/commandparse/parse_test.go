package commandparse

import (
	"reflect"
	"testing"
)

func TestSplit(t *testing.T) {
	got, err := Split(`grep -E 'sword|shield' "loot file.txt"`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"grep", "-E", "sword|shield", "loot file.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}
func TestRejectsShellSyntax(t *testing.T) {
	for _, input := range []string{"grep x f;id", "grep x f | sh", "grep $(id) f", "grep x > out"} {
		if _, err := Split(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestDoubleQuotedRegexKeepsNonSpecialBackslash(t *testing.T) {
	got, err := Split(`grep "\.conf$" files.txt`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"grep", `\.conf$`, "files.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}
