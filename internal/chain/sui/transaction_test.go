package sui

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/timfewi/grepquest/internal/game"
	"github.com/timfewi/grepquest/internal/proof"
)

type recordingRunner struct {
	calls int
	name  string
	args  []string
}

func (r *recordingRunner) Run(_ context.Context, name string, args []string, _, _ io.Writer) error {
	r.calls++
	r.name = name
	r.args = append([]string(nil), args...)
	return nil
}

func TestExecuteRequiresExactConfirmationBeforeWalletCall(t *testing.T) {
	t.Parallel()
	plan := validSubmitPlan(t)
	runner := &recordingRunner{}
	if err := Execute(context.Background(), plan, "yes", runner, io.Discard, io.Discard); err == nil {
		t.Fatal("expected confirmation failure")
	}
	if runner.calls != 0 {
		t.Fatal("wallet was invoked without exact confirmation")
	}
	if err := Execute(context.Background(), plan, plan.Confirmation(), runner, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if runner.calls != 1 || runner.name != "sui-test" || len(runner.args) == 0 {
		t.Fatalf("unexpected invocation: %#v", runner)
	}
}

func TestSubmitPlanRejectsTamperedReceipt(t *testing.T) {
	t.Parallel()
	plan := validSubmitPlan(t)
	plan.Receipt.Score++
	runner := &recordingRunner{}
	if err := Execute(context.Background(), plan, plan.Confirmation(), runner, io.Discard, io.Discard); err == nil {
		t.Fatal("expected tampered receipt to fail")
	}
	if runner.calls != 0 {
		t.Fatal("wallet was invoked with a tampered receipt")
	}
}

func TestSubmitPlanRejectsPlayerClaimedVerifiedTier(t *testing.T) {
	t.Parallel()
	plan := validSubmitPlanWithTier(t, "verified")
	runner := &recordingRunner{}
	if err := Execute(context.Background(), plan, plan.Confirmation(), runner, io.Discard, io.Discard); err == nil {
		t.Fatal("expected a player-claimed verified tier to fail")
	}
	if runner.calls != 0 {
		t.Fatal("wallet was invoked for a player-claimed verified tier")
	}
}

func TestMasteryBitmapIsStable(t *testing.T) {
	t.Parallel()
	first := MasteryBitmap(map[string]int{"literal": 80, "ere": 90})
	second := MasteryBitmap(map[string]int{"ere": 90, "literal": 80})
	if string(first) != string(second) || len(first) != 4 {
		t.Fatalf("unstable bitmap: %v vs %v", first, second)
	}
}

func TestWalletEnvironmentExcludesUnrelatedSecrets(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "must-not-leak")
	t.Setenv("GITHUB_TOKEN", "must-not-leak")
	for _, variable := range CommandEnvironment() {
		if strings.HasPrefix(variable, "OPENROUTER_API_KEY=") || strings.HasPrefix(variable, "GITHUB_TOKEN=") {
			t.Fatalf("secret leaked to wallet process: %s", variable)
		}
	}
}

func validSubmitPlan(t *testing.T) SubmitPlan {
	return validSubmitPlanWithTier(t, "self")
}

func validSubmitPlanWithTier(t *testing.T, tier string) SubmitPlan {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	profile := game.NewProfile()
	profile.XP = 900
	profile.Mastery["literal"] = 90
	receipt, err := proof.NewReceipt(profile, public, proof.Metadata{SeasonID: "core-v1", PackSHA256: sixtyFour("a"), RulesSHA256: sixtyFour("b"), ClientBuildSHA256: sixtyFour("c"), CreatedAt: time.Unix(1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	receipt.ProofTier = tier
	if err = receipt.Sign(private); err != nil {
		t.Fatal(err)
	}
	return SubmitPlan{SuiBinary: "sui-test", Network: "testnet", WalletAddress: "0xa", PackageID: "0x1", ProfileID: "0x2", SeasonID: "0x3", Receipt: receipt, Accuracy: 90, Recall: 80, GasBudget: 10_000_000}
}

func sixtyFour(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return result
}
