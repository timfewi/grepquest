package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/timfewi/grepquest/internal/ai"
	"github.com/timfewi/grepquest/internal/ai/openrouter"
	"github.com/timfewi/grepquest/internal/badge"
	"github.com/timfewi/grepquest/internal/buildinfo"
	"github.com/timfewi/grepquest/internal/challenge"
	"github.com/timfewi/grepquest/internal/game"
	"github.com/timfewi/grepquest/internal/judge"
	"github.com/timfewi/grepquest/internal/learning"
	"github.com/timfewi/grepquest/internal/proof"
	"github.com/timfewi/grepquest/internal/sandbox"
	"github.com/timfewi/grepquest/internal/sandbox/firecracker"
	"github.com/timfewi/grepquest/internal/sandbox/native"
	"github.com/timfewi/grepquest/internal/storage"
	"github.com/timfewi/grepquest/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "grepquest:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "play":
		return play(args[1:])
	case "challenges":
		for _, c := range challenge.Core() {
			fmt.Printf("%-18s Act %d  %s\n", c.ID, c.Act, c.Title)
		}
		return nil
	case "badge":
		return badgeCommand(args[1:])
	case "ai":
		return aiCommand(args[1:])
	case "receipt":
		return receiptCommand(args[1:])
	case "sui":
		return suiCommand(args[1:])
	case "version", "--version":
		fmt.Printf("grepquest %s\n", buildinfo.Version())
		return nil
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() error {
	fmt.Println("GrepQuest — proof-of-skill roguelite for search mastery\n\nUsage:\n  grepquest version\n  grepquest play\n  grepquest challenges\n  grepquest ai enable\n  grepquest ai preview\n  grepquest badge render [--out DIR]\n  grepquest receipt verify FILE\n  grepquest sui leaderboard [OPTIONS]\n  grepquest sui submit [OPTIONS]")
	return nil
}

func play(args []string) error {
	flags := flag.NewFlagSet("play", flag.ContinueOnError)
	backendName := flags.String("sandbox", "native", "sandbox backend: native or firecracker")
	fcBinary := flags.String("firecracker-binary", os.Getenv("GREPQUEST_FIRECRACKER"), "Firecracker executable")
	kernel := flags.String("kernel", os.Getenv("GREPQUEST_KERNEL"), "guest kernel")
	initrd := flags.String("initrd", os.Getenv("GREPQUEST_INITRD"), "guest initrd")
	rootfs := flags.String("rootfs", os.Getenv("GREPQUEST_ROOTFS"), "guest rootfs")
	kernelHash := flags.String("kernel-sha256", os.Getenv("GREPQUEST_KERNEL_SHA256"), "guest kernel digest")
	initrdHash := flags.String("initrd-sha256", os.Getenv("GREPQUEST_INITRD_SHA256"), "guest initrd digest")
	rootfsHash := flags.String("rootfs-sha256", os.Getenv("GREPQUEST_ROOTFS_SHA256"), "guest rootfs digest")
	if err := flags.Parse(args); err != nil {
		return err
	}
	store, profile, err := openProfile()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	var runner sandbox.Sandbox
	switch *backendName {
	case "native":
		fmt.Fprintln(os.Stderr, "warning: native sandbox is for trusted local development only")
		runner = native.New()
	case "firecracker":
		runner = firecracker.New(firecracker.Config{Binary: *fcBinary, Kernel: *kernel, Initrd: *initrd, RootFS: *rootfs, KernelSHA256: *kernelHash, InitrdSHA256: *initrdHash, RootFSSHA256: *rootfsHash, VCPU: 1, MemoryMiB: 256, BootTimeout: 8 * time.Second, RunTimeout: 2 * time.Second})
	default:
		return fmt.Errorf("unknown sandbox %q", *backendName)
	}
	content := game.ContentGenerator(ai.Fallback{})
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		client, clientErr := openrouter.New(openrouter.Config{APIKey: key, Model: os.Getenv("GREPQUEST_AI_MODEL"), Cache: store, MaxSessionRequests: 10, MaxDailyRequests: 30})
		if clientErr != nil {
			return clientErr
		}
		content = ai.WithFallback{Primary: client, Fallback: ai.Fallback{}}
	}
	engine := game.Engine{Judge: judge.Judge{Sandbox: runner}, Content: content}
	model := tui.New(engine, learning.Queue(challenge.Core(), profile, time.Now().UTC()), profile, func(p game.Profile) error { return store.Save(context.Background(), p) })
	_, err = tea.NewProgram(model).Run()
	return err
}

func aiCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: grepquest ai enable|preview")
	}
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return openrouter.ErrDisabled
	}
	switch args[0] {
	case "enable":
		fmt.Println("OpenRouter is enabled for this process; the key remains runtime-only and is not stored.")
		return nil
	case "preview":
		store, _, err := openProfile()
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()
		client, err := openrouter.New(openrouter.Config{APIKey: key, Model: os.Getenv("GREPQUEST_AI_MODEL"), Cache: store})
		if err != nil {
			return err
		}
		item := game.BuiltinItems[0]
		content, err := client.Generate(context.Background(), game.ContentRequest{Item: item, ChallengeID: "preview", ChallengeTitle: "The Signal Cellar", Skill: item.Skill})
		if err != nil {
			return err
		}
		fmt.Printf("%s %s\n%s\n", content.Sprite, content.Name, content.Flavor)
		return nil
	default:
		return errors.New("usage: grepquest ai enable|preview")
	}
}

func badgeCommand(args []string) error {
	if len(args) == 0 || args[0] != "render" {
		return errors.New("usage: grepquest badge render [--out DIR]")
	}
	flags := flag.NewFlagSet("badge render", flag.ContinueOnError)
	out := flags.String("out", ".", "output directory")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	store, profile, err := openProfile()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	manifest, err := challenge.CoreManifest("GNU grep 3.12")
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	buildHash, err := proof.SHA256File(executable)
	if err != nil {
		return err
	}
	rules := sha256.Sum256([]byte("grepquest-score-v1:base=100+difficulty*20;hint=-10;level=1+xp/300"))
	statePath, err := stateDir()
	if err != nil {
		return err
	}
	key, err := proof.LoadOrCreateKey(filepath.Join(statePath, "signing-key"))
	if err != nil {
		return err
	}
	receipt, err := proof.NewReceipt(profile, key.Public().(ed25519.PublicKey), proof.Metadata{SeasonID: "core-v1", PackSHA256: manifest.PackSHA256, RulesSHA256: hex.EncodeToString(rules[:]), ClientBuildSHA256: buildHash, CreatedAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	if err = receipt.Sign(key); err != nil {
		return err
	}
	receipt, err = badge.Render(*out, profile, receipt)
	if err != nil {
		return err
	}
	fmt.Printf("rendered grepquest.svg and grepquest-proof.json (receipt %s)\n", receipt.ReceiptSHA256[:8])
	return nil
}

func receiptCommand(args []string) error {
	if len(args) != 2 || args[0] != "verify" {
		return errors.New("usage: grepquest receipt verify FILE")
	}
	file, err := os.Open(args[1])
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var receipt proof.Receipt
	if err = decoder.Decode(&receipt); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("receipt contains trailing JSON")
	}
	if err = receipt.Verify(); err != nil {
		return err
	}
	fmt.Printf("valid %s receipt %s for %s (score %d)\n", receipt.ProofTier, receipt.ReceiptSHA256[:8], receipt.SeasonID, receipt.Score)
	return nil
}

func openProfile() (*storage.Store, game.Profile, error) {
	statePath, err := stateDir()
	if err != nil {
		return nil, game.Profile{}, err
	}
	store, err := storage.Open(filepath.Join(statePath, "grepquest.db"))
	if err != nil {
		return nil, game.Profile{}, err
	}
	profile, err := store.Load(context.Background())
	if err != nil {
		_ = store.Close()
		return nil, profile, err
	}
	return store, profile, nil
}

func stateDir() (string, error) {
	if state := os.Getenv("XDG_STATE_HOME"); state != "" {
		return filepath.Join(state, "grepquest"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "grepquest"), nil
}
