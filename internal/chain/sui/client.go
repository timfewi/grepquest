package sui

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	pageSize         = 50
	maxResponseBytes = 1 << 20
	maxQueryPages    = 20
)

// GraphQLClient reads public ScoreSubmitted events. It has no wallet or write
// capability; ranking stays a local operation.
type GraphQLClient struct {
	endpoint string
	http     *http.Client
}

type ScoreEvent struct {
	Player        string
	SeasonID      string
	Score         uint64
	Level         uint64
	ReceiptSHA256 string
	ProofTier     ProofTier
	Accuracy      uint64
	Recall        uint64
	NoHintClears  uint64
	ElapsedMS     uint64
	Timestamp     string
}

type ProofTier uint8

const (
	SelfAttested ProofTier = iota
	Verified
	Official
)

func (p ProofTier) String() string {
	switch p {
	case SelfAttested:
		return "self"
	case Verified:
		return "verified"
	case Official:
		return "official"
	default:
		return "unknown"
	}
}

func ParseProofTier(value string) (ProofTier, error) {
	switch value {
	case "self":
		return SelfAttested, nil
	case "verified":
		return Verified, nil
	case "official":
		return Official, nil
	default:
		return 0, fmt.Errorf("unknown proof tier %q", value)
	}
}

func NewGraphQLClient(endpoint string, base *http.Client) (*GraphQLClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid Sui GraphQL endpoint")
	}
	host := u.Hostname()
	if u.Scheme != "https" && (u.Scheme != "http" || !isLoopback(host)) {
		return nil, errors.New("sui GraphQL endpoint must use HTTPS (HTTP is allowed only on loopback)")
	}
	client := http.Client{Timeout: 10 * time.Second}
	if base != nil {
		client = *base
		if client.Timeout == 0 {
			client.Timeout = 10 * time.Second
		}
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &GraphQLClient{endpoint: u.String(), http: &client}, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func EventType(packageID string) (string, error) {
	if err := validateSuiID(packageID); err != nil {
		return "", err
	}
	return packageID + "::social::ScoreSubmitted", nil
}

// FetchScoreEvents follows at most maxPages bounded GraphQL pages. The caller
// selects the package by its fully-qualified event type.
func (c *GraphQLClient) FetchScoreEvents(ctx context.Context, eventType string, maxPages int) ([]ScoreEvent, error) {
	if c == nil || c.http == nil {
		return nil, errors.New("sui GraphQL client is required")
	}
	if maxPages <= 0 || maxPages > maxQueryPages {
		return nil, fmt.Errorf("max pages must be between 1 and %d", maxQueryPages)
	}
	if !validEventType(eventType) {
		return nil, errors.New("invalid ScoreSubmitted event type")
	}
	var cursor any
	result := make([]ScoreEvent, 0)
	for pageNumber := 0; pageNumber < maxPages; pageNumber++ {
		page, err := c.fetchPage(ctx, eventType, cursor)
		if err != nil {
			return nil, err
		}
		for _, node := range page.Nodes {
			event, err := parseScoreEvent(node)
			if err != nil {
				return nil, fmt.Errorf("invalid ScoreSubmitted event: %w", err)
			}
			result = append(result, event)
		}
		if !page.PageInfo.HasNextPage {
			return result, nil
		}
		if page.PageInfo.EndCursor == "" {
			return nil, errors.New("GraphQL response has next page without a cursor")
		}
		cursor = page.PageInfo.EndCursor
	}
	return nil, errors.New("sui GraphQL page limit reached; ranking would be incomplete")
}

const eventsQuery = `query ScoreEvents($eventType: String!, $first: Int!, $after: String) {
  events(filter: { type: $eventType }, first: $first, after: $after) {
    nodes { sender { address } timestamp contents { json } }
    pageInfo { hasNextPage endCursor }
  }
}`

type eventNode struct {
	Sender struct {
		Address string `json:"address"`
	} `json:"sender"`
	Timestamp string `json:"timestamp"`
	Contents  struct {
		JSON json.RawMessage `json:"json"`
	} `json:"contents"`
}

type eventPage struct {
	Nodes    []eventNode `json:"nodes"`
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
}

func (c *GraphQLClient) fetchPage(ctx context.Context, eventType string, cursor any) (eventPage, error) {
	payload := struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}{Query: eventsQuery, Variables: map[string]any{"eventType": eventType, "first": pageSize, "after": cursor}}
	body, err := json.Marshal(payload)
	if err != nil {
		return eventPage{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return eventPage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return eventPage{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return eventPage{}, fmt.Errorf("sui GraphQL returned HTTP %d", resp.StatusCode)
	}
	rawResponse, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return eventPage{}, fmt.Errorf("read Sui GraphQL response: %w", err)
	}
	if len(rawResponse) > maxResponseBytes {
		return eventPage{}, errors.New("sui GraphQL response exceeds 1 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(rawResponse))
	var envelope struct {
		Data struct {
			Events eventPage `json:"events"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err = decoder.Decode(&envelope); err != nil {
		return eventPage{}, fmt.Errorf("decode Sui GraphQL response: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return eventPage{}, errors.New("sui GraphQL response contains trailing JSON")
	}
	if len(envelope.Errors) > 0 {
		return eventPage{}, fmt.Errorf("sui GraphQL returned %d error(s)", len(envelope.Errors))
	}
	return envelope.Data.Events, nil
}

func validEventType(value string) bool {
	parts := strings.Split(value, "::")
	if len(parts) != 3 || parts[1] != "social" || parts[2] != "ScoreSubmitted" {
		return false
	}
	return validateSuiID(parts[0]) == nil
}

func parseScoreEvent(node eventNode) (ScoreEvent, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(node.Contents.JSON, &raw); err != nil {
		return ScoreEvent{}, err
	}
	player, err := textField(raw, "player")
	if err != nil {
		return ScoreEvent{}, err
	}
	if player == "" {
		player = node.Sender.Address
	}
	if err = validateSuiID(player); err != nil {
		return ScoreEvent{}, errors.New("invalid player address")
	}
	season, err := textField(raw, "season_id")
	if err != nil {
		return ScoreEvent{}, err
	}
	if err = ValidateSeasonID(season); err != nil {
		return ScoreEvent{}, err
	}
	receipt, err := digestField(raw, "receipt_sha256")
	if err != nil {
		return ScoreEvent{}, err
	}
	values := make(map[string]uint64, 7)
	for _, field := range []string{"score", "level", "proof_tier", "accuracy", "recall", "no_hint_clears", "elapsed_ms"} {
		values[field], err = uintField(raw, field)
		if err != nil {
			return ScoreEvent{}, err
		}
	}
	if values["proof_tier"] > uint64(Official) || values["accuracy"] > 100 || values["recall"] > 100 {
		return ScoreEvent{}, errors.New("event metric or proof tier is out of range")
	}
	return ScoreEvent{Player: player, SeasonID: season, Score: values["score"], Level: values["level"], ReceiptSHA256: receipt, ProofTier: ProofTier(values["proof_tier"]), Accuracy: values["accuracy"], Recall: values["recall"], NoHintClears: values["no_hint_clears"], ElapsedMS: values["elapsed_ms"], Timestamp: node.Timestamp}, nil
}

func digestField(raw map[string]json.RawMessage, name string) (string, error) {
	value, ok := raw[name]
	if !ok {
		return "", fmt.Errorf("missing %s", name)
	}
	var text string
	if json.Unmarshal(value, &text) == nil {
		text = strings.TrimPrefix(text, "0x")
		if len(text) == 64 && hexText(text) {
			return strings.ToLower(text), nil
		}
	}
	var bytesValue []byte
	if err := json.Unmarshal(value, &bytesValue); err == nil && len(bytesValue) == 32 {
		return hex.EncodeToString(bytesValue), nil
	}
	return "", fmt.Errorf("invalid %s", name)
}

func ValidateSeasonID(value string) error {
	if value == "" || len(value) > 128 {
		return errors.New("season ID must contain 1-128 printable ASCII bytes")
	}
	for _, char := range value {
		if char < 0x20 || char > 0x7e {
			return errors.New("season ID must contain 1-128 printable ASCII bytes")
		}
	}
	return nil
}

func hexText(value string) bool {
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9':
		case char >= 'a' && char <= 'f':
		case char >= 'A' && char <= 'F':
		default:
			return false
		}
	}
	return true
}

func textField(raw map[string]json.RawMessage, name string) (string, error) {
	value, ok := raw[name]
	if !ok {
		return "", fmt.Errorf("missing %s", name)
	}
	var text string
	if json.Unmarshal(value, &text) == nil {
		return text, nil
	}
	var bytesValue []byte
	if err := json.Unmarshal(value, &bytesValue); err == nil {
		return string(bytesValue), nil
	}
	return "", fmt.Errorf("invalid %s", name)
}

func uintField(raw map[string]json.RawMessage, name string) (uint64, error) {
	value, ok := raw[name]
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	var number uint64
	if json.Unmarshal(value, &number) == nil {
		return number, nil
	}
	var text string
	if json.Unmarshal(value, &text) == nil {
		number, err := strconv.ParseUint(text, 10, 64)
		if err == nil {
			return number, nil
		}
	}
	return 0, fmt.Errorf("invalid %s", name)
}
