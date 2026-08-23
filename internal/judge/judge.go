package judge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/sandbox"
)

type Verdict struct {
	Correct                      bool
	ExpectedOutput, ActualOutput string
	ExpectedExit, ActualExit     int
	Explanation                  string
	Duration                     time.Duration
	HiddenPassed, HiddenTotal    int
}

type Judge struct{ Sandbox sandbox.Sandbox }

func (j Judge) Evaluate(ctx context.Context, c challenge.Challenge, command string) (Verdict, error) {
	if j.Sandbox == nil {
		return Verdict{}, fmt.Errorf("sandbox is required")
	}
	if err := j.Sandbox.Boot(ctx, c); err != nil {
		return Verdict{}, fmt.Errorf("boot sandbox: %w", err)
	}
	defer func() { _ = j.Sandbox.Close(context.Background()) }()
	result, err := j.Sandbox.Run(ctx, command)
	verdict := Verdict{ExpectedOutput: c.WantOutput, ActualOutput: result.Stdout, ExpectedExit: c.WantExit, ActualExit: result.ExitCode, Explanation: c.Explanation, Duration: result.Duration}
	if err != nil {
		return verdict, fmt.Errorf("run command: %w", err)
	}
	if !containsAll(result.Args, c.RequiredArgs) {
		verdict.Explanation = "The output matched, but this room requires practicing: " + strings.Join(c.RequiredArgs, ", ")
		return verdict, nil
	}
	verdict.Correct = normalize(result.Stdout) == normalize(c.WantOutput) && result.ExitCode == c.WantExit
	if !verdict.Correct {
		return verdict, nil
	}
	verdict.HiddenTotal = len(c.HiddenTests)
	for _, test := range c.HiddenTests {
		variant := c
		variant.Fixtures = test.Fixtures
		variant.WantOutput = test.WantOutput
		variant.WantExit = test.WantExit
		variant.HiddenTests = nil
		if err := j.Sandbox.Boot(ctx, variant); err != nil {
			return verdict, fmt.Errorf("boot hidden test: %w", err)
		}
		hidden, err := j.Sandbox.Run(ctx, command)
		if err != nil {
			return verdict, fmt.Errorf("run hidden test: %w", err)
		}
		if normalize(hidden.Stdout) == normalize(test.WantOutput) && hidden.ExitCode == test.WantExit {
			verdict.HiddenPassed++
		} else {
			verdict.Correct = false
		}
	}
	return verdict, nil
}

func normalize(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
func containsAll(args, required []string) bool {
	for _, want := range required {
		found := false
		for _, got := range args {
			if got == want || (len(want) == 2 && want[0] == '-' && len(got) > 2 && got[0] == '-' && !strings.HasPrefix(got, "--") && strings.ContainsRune(got, rune(want[1]))) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
