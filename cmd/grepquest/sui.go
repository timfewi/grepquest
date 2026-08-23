package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	chainsui "github.com/timfewi/grepquest/internal/chain/sui"
	"github.com/timfewi/grepquest/internal/proof"
)

func suiCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: grepquest sui leaderboard|submit")
	}
	switch args[0] {
	case "leaderboard":
		return suiLeaderboard(args[1:])
	case "submit":
		return suiSubmit(args[1:])
	default:
		return errors.New("usage: grepquest sui leaderboard|submit")
	}
}

func suiLeaderboard(args []string) error {
	flags := flag.NewFlagSet("sui leaderboard", flag.ContinueOnError)
	endpoint := flags.String("graphql", "https://graphql.testnet.sui.io/graphql", "Sui GraphQL endpoint")
	packageID := flags.String("package", "", "published GrepQuest package ID")
	season := flags.String("season", "", "season ID")
	tierName := flags.String("tier", "self", "proof tier: self, verified, or official")
	metricName := flags.String("metric", "score", "metric: score, accuracy, recall, no-hint, or speed")
	pages := flags.Int("pages", 4, "pages to read (1-20, 50 events each)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *packageID == "" || *season == "" {
		return errors.New("--package and --season are required")
	}
	if err := chainsui.ValidateSeasonID(*season); err != nil {
		return err
	}
	tier, err := chainsui.ParseProofTier(*tierName)
	if err != nil {
		return err
	}
	metric, err := chainsui.ParseMetric(*metricName)
	if err != nil {
		return err
	}
	eventType, err := chainsui.EventType(*packageID)
	if err != nil {
		return err
	}
	client, err := chainsui.NewGraphQLClient(*endpoint, nil)
	if err != nil {
		return err
	}
	events, err := client.FetchScoreEvents(context.Background(), eventType, *pages)
	if err != nil {
		return err
	}
	ranking := chainsui.Rank(events, *season, tier, metric)
	fmt.Printf("%s · %s · %s\n", *season, tier, metric)
	for _, entry := range ranking {
		fmt.Printf("%3d  %-66s  %d  receipt %.8s\n", entry.Rank, entry.Event.Player, entry.Value, entry.Event.ReceiptSHA256)
	}
	return nil
}

func suiSubmit(args []string) error {
	flags := flag.NewFlagSet("sui submit", flag.ContinueOnError)
	receiptPath := flags.String("receipt", "", "signed proof receipt")
	packageID := flags.String("package", "", "published GrepQuest package ID")
	profileID := flags.String("profile", "", "address-owned PlayerProfile object ID")
	seasonObjectID := flags.String("season-object", "", "immutable Season object ID")
	walletAddress := flags.String("wallet", "", "expected active Sui CLI wallet address")
	network := flags.String("network", "testnet", "expected active Sui CLI environment")
	suiBinary := flags.String("sui-binary", "sui", "Sui CLI executable")
	gasBudget := flags.Uint64("gas-budget", 10_000_000, "maximum gas budget in MIST")
	accuracy := flags.Uint64("accuracy", 0, "self-attested accuracy percentage")
	recall := flags.Uint64("recall", 0, "self-attested recall percentage")
	noHint := flags.Uint64("no-hint", 0, "self-attested no-hint clears")
	elapsed := flags.Uint64("elapsed-ms", 0, "hardware-dependent speed metric")
	execute := flags.Bool("execute", false, "sign and submit after exact interactive confirmation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *receiptPath == "" || *packageID == "" || *profileID == "" || *seasonObjectID == "" || *walletAddress == "" {
		return errors.New("--receipt, --package, --profile, --season-object, and --wallet are required")
	}
	receipt, err := loadReceipt(*receiptPath)
	if err != nil {
		return err
	}
	plan := chainsui.SubmitPlan{SuiBinary: *suiBinary, Network: *network, WalletAddress: *walletAddress, PackageID: *packageID, ProfileID: *profileID, SeasonID: *seasonObjectID, Receipt: receipt, Accuracy: *accuracy, Recall: *recall, NoHintClears: *noHint, ElapsedMS: *elapsed, GasBudget: *gasBudget}
	if err = plan.Validate(); err != nil {
		return err
	}
	fmt.Println(plan.Summary())
	if !*execute {
		fmt.Println("preview only: no wallet was opened and no transaction was submitted; add --execute to continue")
		return nil
	}
	activeNetwork, err := activeSuiSetting(context.Background(), *suiBinary, "active-env")
	if err != nil {
		return err
	}
	if activeNetwork != *network {
		return fmt.Errorf("active Sui environment %q does not match confirmed network %q", activeNetwork, *network)
	}
	activeWallet, err := activeSuiSetting(context.Background(), *suiBinary, "active-address")
	if err != nil {
		return err
	}
	if _, err = chainsui.EventType(activeWallet); err != nil {
		return errors.New("sui CLI returned an invalid active wallet address")
	}
	if !strings.EqualFold(activeWallet, *walletAddress) {
		return fmt.Errorf("active Sui wallet %s does not match confirmed wallet %s", activeWallet, *walletAddress)
	}
	fmt.Printf("Type %q to ask wallet %s to sign and submit: ", plan.Confirmation(), activeWallet)
	confirmation, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return chainsui.Execute(context.Background(), plan, strings.TrimSpace(confirmation), chainsui.ExecRunner{}, os.Stdout, os.Stderr)
}

func activeSuiSetting(ctx context.Context, binary, setting string) (string, error) {
	command := exec.CommandContext(ctx, binary, "client", setting)
	command.Env = chainsui.CommandEnvironment()
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("read Sui CLI %s: %w", setting, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func loadReceipt(path string) (proof.Receipt, error) {
	file, err := os.Open(path)
	if err != nil {
		return proof.Receipt{}, err
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return proof.Receipt{}, err
	}
	if len(raw) > 1<<20 {
		return proof.Receipt{}, errors.New("receipt exceeds 1 MiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var receipt proof.Receipt
	if err = decoder.Decode(&receipt); err != nil {
		return proof.Receipt{}, err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return proof.Receipt{}, errors.New("receipt contains trailing JSON")
	}
	if err = receipt.Verify(); err != nil {
		return proof.Receipt{}, err
	}
	return receipt, nil
}
