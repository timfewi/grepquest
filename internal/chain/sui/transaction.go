package sui

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/timfewi/grepquest/internal/proof"
)

type SubmitPlan struct {
	SuiBinary     string
	Network       string
	WalletAddress string
	PackageID     string
	ProfileID     string
	SeasonID      string
	Receipt       proof.Receipt
	Accuracy      uint64
	Recall        uint64
	NoHintClears  uint64
	ElapsedMS     uint64
	GasBudget     uint64
}

type Runner interface {
	Run(context.Context, string, []string, io.Writer, io.Writer) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = CommandEnvironment()
	return cmd.Run()
}

// CommandEnvironment permits wallet configuration and terminal basics while
// excluding unrelated host secrets such as OPENROUTER_API_KEY and Git tokens.
func CommandEnvironment() []string {
	keys := []string{"HOME", "PATH", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "LANG", "LC_ALL", "TERM", "NO_COLOR", "SSL_CERT_FILE", "NIX_SSL_CERT_FILE"}
	environment := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			environment = append(environment, key+"="+value)
		}
	}
	return environment
}

func (p SubmitPlan) Validate() error {
	if p.SuiBinary == "" {
		return errors.New("sui CLI executable is required")
	}
	if !safeNetworkName(p.Network) {
		return errors.New("a safe expected Sui network name is required")
	}
	for _, item := range []struct{ label, value string }{{"wallet", p.WalletAddress}, {"package", p.PackageID}, {"profile", p.ProfileID}, {"season", p.SeasonID}} {
		if err := validateSuiID(item.value); err != nil {
			return fmt.Errorf("invalid %s address or object ID", item.label)
		}
	}
	if err := p.Receipt.Verify(); err != nil {
		return fmt.Errorf("receipt verification failed: %w", err)
	}
	if err := ValidateSeasonID(p.Receipt.SeasonID); err != nil {
		return fmt.Errorf("invalid receipt season ID: %w", err)
	}
	for _, item := range []struct{ label, digest string }{{"pack", p.Receipt.PackSHA256}, {"rules", p.Receipt.RulesSHA256}, {"receipt", p.Receipt.ReceiptSHA256}} {
		if _, err := decodeDigest(item.digest); err != nil {
			return fmt.Errorf("invalid %s SHA-256 digest", item.label)
		}
	}
	if p.Receipt.Score < 0 || p.Receipt.Level() < 1 {
		return errors.New("receipt score and level are invalid")
	}
	if p.Accuracy > 100 || p.Recall > 100 {
		return errors.New("accuracy and recall must be between 0 and 100")
	}
	if _, err := ParseProofTier(p.Receipt.ProofTier); err != nil {
		return err
	}
	if p.Receipt.ProofTier != "self" {
		return errors.New("player wallet submission accepts only self-attested receipts")
	}
	if p.GasBudget == 0 {
		return errors.New("a non-zero gas budget is required")
	}
	return nil
}

func (p SubmitPlan) Confirmation() string {
	prefix := p.Receipt.ReceiptSHA256
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	return "SUBMIT " + p.Network + " " + p.WalletAddress + " " + prefix
}

func (p SubmitPlan) Summary() string {
	return fmt.Sprintf("network: %s\nwallet: %s (external Sui CLI)\npackage: %s\nfunction: social::submit_score\nprofile: %s\nseason object: %s\nreceipt season: %s\npack SHA-256: %s\nrules SHA-256: %s\nscore: %d\nlevel: %d\nproof tier: %s\nreceipt: %s\naccuracy: %d\nrecall: %d\nno-hint clears: %d\nelapsed: %d ms\ngas budget: %d MIST", p.Network, p.WalletAddress, p.PackageID, p.ProfileID, p.SeasonID, p.Receipt.SeasonID, p.Receipt.PackSHA256, p.Receipt.RulesSHA256, p.Receipt.Score, p.Receipt.Level(), p.Receipt.ProofTier, p.Receipt.ReceiptSHA256, p.Accuracy, p.Recall, p.NoHintClears, p.ElapsedMS, p.GasBudget)
}

func safeNetworkName(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case char == '-' || char == '_' || char == '.':
		default:
			return false
		}
	}
	return true
}

func (p SubmitPlan) Arguments() ([]string, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	receiptBytes, _ := decodeDigest(p.Receipt.ReceiptSHA256)
	packBytes, _ := decodeDigest(p.Receipt.PackSHA256)
	rulesBytes, _ := decodeDigest(p.Receipt.RulesSHA256)
	tier, _ := ParseProofTier(p.Receipt.ProofTier)
	args := []string{
		"client", "call",
		"--package", p.PackageID,
		"--module", "social",
		"--function", "submit_score",
		"--args", p.ProfileID, p.SeasonID,
		byteVector([]byte(p.Receipt.SeasonID)),
		byteVector(packBytes),
		byteVector(rulesBytes),
		byteVector(MasteryBitmap(p.Receipt.Mastery)),
		strconv.FormatUint(uint64(p.Receipt.Level()), 10),
		strconv.FormatUint(uint64(p.Receipt.Score), 10),
		byteVector(receiptBytes),
		strconv.FormatUint(uint64(tier), 10),
		strconv.FormatUint(p.Accuracy, 10),
		strconv.FormatUint(p.Recall, 10),
		strconv.FormatUint(p.NoHintClears, 10),
		strconv.FormatUint(p.ElapsedMS, 10),
		"--gas-budget", strconv.FormatUint(p.GasBudget, 10),
	}
	return args, nil
}

func decodeDigest(value string) ([]byte, error) {
	if len(value) != 64 {
		return nil, errors.New("SHA-256 digest must contain 64 hexadecimal characters")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, errors.New("SHA-256 digest is not hexadecimal")
	}
	return decoded, nil
}

// Execute is the only write path. A mismatched confirmation never invokes the
// external wallet, and arguments are passed directly without a shell.
func Execute(ctx context.Context, p SubmitPlan, confirmation string, runner Runner, stdout, stderr io.Writer) error {
	if confirmation != p.Confirmation() {
		return errors.New("transaction was not confirmed")
	}
	if runner == nil {
		return errors.New("transaction runner is required")
	}
	args, err := p.Arguments()
	if err != nil {
		return err
	}
	return runner.Run(ctx, p.SuiBinary, args, stdout, stderr)
}

var masterySkills = []string{
	"-E", "-F", "-e", "-i", "-w", "-x", "alternation", "anchor", "bre", "case", "class", "ere", "error", "escaping", "exit-code", "files", "fixed", "group", "line", "literal", "multiple-pattern", "quantifier", "quoting", "recall", "word",
}

func MasteryBitmap(mastery map[string]int) []byte {
	skills := append([]string(nil), masterySkills...)
	sort.Strings(skills)
	bitmap := make([]byte, (len(skills)+7)/8)
	for index, skill := range skills {
		if mastery[skill] >= 80 {
			bitmap[index/8] |= 1 << uint(index%8)
		}
	}
	return bitmap
}

func byteVector(value []byte) string {
	parts := make([]string, len(value))
	for i, b := range value {
		parts[i] = strconv.Itoa(int(b))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func validateSuiID(value string) error {
	if !strings.HasPrefix(value, "0x") || len(value) < 3 || len(value) > 66 {
		return errors.New("sui ID must be 0x-prefixed with at most 32 bytes")
	}
	for _, char := range strings.TrimPrefix(value, "0x") {
		switch {
		case char >= '0' && char <= '9':
		case char >= 'a' && char <= 'f':
		case char >= 'A' && char <= 'F':
		default:
			return errors.New("sui ID is not hexadecimal")
		}
	}
	return nil
}
