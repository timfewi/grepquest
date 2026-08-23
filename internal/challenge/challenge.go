package challenge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Challenge is the canonical, data-only unit consumed by the game and judge.
type Challenge struct {
	ID           string            `json:"id"`
	Act          int               `json:"act"`
	Title        string            `json:"title"`
	Prompt       string            `json:"prompt"`
	Tags         []string          `json:"tags"`
	Difficulty   int               `json:"difficulty"`
	Fixtures     map[string]string `json:"fixtures"`
	WantOutput   string            `json:"want_output"`
	WantExit     int               `json:"want_exit"`
	Hints        []string          `json:"hints"`
	Explanation  string            `json:"explanation"`
	RequiredArgs []string          `json:"required_args,omitempty"`
	HiddenTests  []TestCase        `json:"hidden_tests,omitempty"`
}

type TestCase struct {
	Fixtures   map[string]string `json:"fixtures"`
	WantOutput string            `json:"want_output"`
	WantExit   int               `json:"want_exit"`
}

func (c Challenge) Validate() error {
	if c.ID == "" || c.Title == "" || c.Prompt == "" {
		return fmt.Errorf("challenge id, title, and prompt are required")
	}
	if c.Act < 1 || c.Act > 2 || c.Difficulty < 1 || c.Difficulty > 5 {
		return fmt.Errorf("challenge %q has invalid act or difficulty", c.ID)
	}
	if c.WantExit < 0 || c.WantExit > 2 {
		return fmt.Errorf("challenge %q has invalid expected exit code", c.ID)
	}
	for name := range c.Fixtures {
		if name == "" || name[0] == '/' || containsDotDot(name) {
			return fmt.Errorf("challenge %q has unsafe fixture path %q", c.ID, name)
		}
	}
	for i, test := range c.HiddenTests {
		if test.WantExit < 0 || test.WantExit > 2 {
			return fmt.Errorf("challenge %q hidden test %d has invalid exit code", c.ID, i)
		}
		for name := range test.Fixtures {
			if name == "" || name[0] == '/' || containsDotDot(name) {
				return fmt.Errorf("challenge %q hidden test %d has unsafe fixture path %q", c.ID, i, name)
			}
		}
	}
	return nil
}

func containsDotDot(path string) bool {
	for _, part := range splitSlash(path) {
		if part == ".." {
			return true
		}
	}
	return false
}

func splitSlash(s string) []string {
	var out []string
	start := 0
	for i, r := range s {
		if r == '/' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// Hash uses a stable wire representation. Maps are converted to sorted pairs
// instead of relying on an encoder's map ordering.
func (c Challenge) Hash() (string, error) {
	type file struct{ Name, Contents string }
	type canonical struct {
		ID            string
		Act           int
		Title, Prompt string
		Tags          []string
		Difficulty    int
		Fixtures      []file
		WantOutput    string
		WantExit      int
		Hints         []string
		Explanation   string
		RequiredArgs  []string
		HiddenTests   []TestCase
	}
	files := make([]file, 0, len(c.Fixtures))
	for name, contents := range c.Fixtures {
		files = append(files, file{name, contents})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	b, err := json.Marshal(canonical{c.ID, c.Act, c.Title, c.Prompt, c.Tags, c.Difficulty, files, c.WantOutput, c.WantExit, c.Hints, c.Explanation, c.RequiredArgs, c.HiddenTests})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
