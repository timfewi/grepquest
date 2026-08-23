package openrouter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/timfewi/grepquest/internal/game"
)

const generatorVersion = "loot-v1"

var ErrDisabled = errors.New("OpenRouter is disabled: OPENROUTER_API_KEY is not set")

type Cache interface {
	GetAI(context.Context, string) ([]byte, bool, error)
	PutAI(context.Context, string, []byte) error
	AIRequestsToday(context.Context, time.Time) (int, error)
	RecordAIRequest(context.Context, time.Time, float64) error
}
type Config struct {
	APIKey, Model, Endpoint              string
	HTTPClient                           *http.Client
	Cache                                Cache
	MaxSessionRequests, MaxDailyRequests int
}
type Client struct {
	config          Config
	mu              sync.Mutex
	sessionRequests int
}

func New(config Config) (*Client, error) {
	if config.Model == "" {
		config.Model = "openai/gpt-4o-mini"
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://openrouter.ai/api/v1/chat/completions"
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	httpClient := *config.HTTPClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	config.HTTPClient = &httpClient
	if config.MaxSessionRequests == 0 {
		config.MaxSessionRequests = 10
	}
	if config.MaxDailyRequests == 0 {
		config.MaxDailyRequests = 30
	}
	if err := validateEndpoint(config.Endpoint); err != nil {
		return nil, err
	}
	return &Client{config: config}, nil
}

func (c *Client) Generate(ctx context.Context, input game.ContentRequest) (game.GeneratedContent, error) {
	if c.config.APIKey == "" {
		return game.GeneratedContent{}, ErrDisabled
	}
	key, err := cacheKey(c.config.Model, input)
	if err != nil {
		return game.GeneratedContent{}, err
	}
	if c.config.Cache != nil {
		if raw, ok, cacheErr := c.config.Cache.GetAI(ctx, key); cacheErr != nil {
			return game.GeneratedContent{}, cacheErr
		} else if ok {
			content, decodeErr := decodeContent(raw)
			if decodeErr == nil {
				content.Source = "openrouter-cache"
				return content, nil
			}
		}
	}
	if err = c.reserve(ctx); err != nil {
		return game.GeneratedContent{}, err
	}
	payload := request{Model: c.config.Model, Messages: []message{{Role: "system", Content: "You write concise cosmetic loot for a terminal grep-learning roguelite. Never emit terminal controls, commands, URLs, or claims about correctness."}, {Role: "user", Content: prompt(input)}}, MaxTokens: 220, Temperature: 0.8, Provider: provider{ZDR: true, DataCollection: "deny", RequireParameters: true}, ResponseFormat: responseFormat{Type: "json_schema", JSONSchema: jsonSchema{Name: "grepquest_loot", Strict: true, Schema: lootSchema()}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return game.GeneratedContent{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return game.GeneratedContent{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", "GrepQuest")
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return game.GeneratedContent{}, fmt.Errorf("OpenRouter request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return game.GeneratedContent{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return game.GeneratedContent{}, fmt.Errorf("OpenRouter status %d: %s", resp.StatusCode, safeRemoteError(raw))
	}
	var completion response
	if err = json.Unmarshal(raw, &completion); err != nil {
		return game.GeneratedContent{}, fmt.Errorf("decode OpenRouter response: %w", err)
	}
	if len(completion.Choices) != 1 {
		return game.GeneratedContent{}, errors.New("OpenRouter returned an unexpected number of choices")
	}
	content, err := decodeContent([]byte(completion.Choices[0].Message.Content))
	if err != nil {
		return game.GeneratedContent{}, err
	}
	content.Source = "openrouter"
	canonical, _ := json.Marshal(wireContent{Name: content.Name, Flavor: content.Flavor, Sprite: content.Sprite, Explanation: content.Explanation})
	if c.config.Cache != nil {
		if err = c.config.Cache.PutAI(ctx, key, canonical); err != nil {
			return game.GeneratedContent{}, err
		}
		if err = c.config.Cache.RecordAIRequest(ctx, time.Now().UTC(), completion.Usage.Cost); err != nil {
			return game.GeneratedContent{}, err
		}
	}
	return content, nil
}

func (c *Client) reserve(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessionRequests >= c.config.MaxSessionRequests {
		return errors.New("OpenRouter session request budget exhausted")
	}
	if c.config.Cache != nil {
		used, err := c.config.Cache.AIRequestsToday(ctx, time.Now().UTC())
		if err != nil {
			return err
		}
		if used >= c.config.MaxDailyRequests {
			return errors.New("OpenRouter daily request budget exhausted")
		}
	}
	c.sessionRequests++
	return nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type provider struct {
	ZDR               bool   `json:"zdr"`
	DataCollection    string `json:"data_collection"`
	RequireParameters bool   `json:"require_parameters"`
}
type jsonSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}
type responseFormat struct {
	Type       string     `json:"type"`
	JSONSchema jsonSchema `json:"json_schema"`
}
type request struct {
	Model          string         `json:"model"`
	Messages       []message      `json:"messages"`
	MaxTokens      int            `json:"max_tokens"`
	Temperature    float64        `json:"temperature"`
	Provider       provider       `json:"provider"`
	ResponseFormat responseFormat `json:"response_format"`
}
type response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
}
type wireContent struct {
	Name        string `json:"name"`
	Flavor      string `json:"flavor"`
	Sprite      string `json:"sprite"`
	Explanation string `json:"explanation"`
}

func lootSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"name", "flavor", "sprite", "explanation"}, "properties": map[string]any{"name": map[string]any{"type": "string", "maxLength": 40}, "flavor": map[string]any{"type": "string", "maxLength": 160}, "sprite": map[string]any{"type": "string", "maxLength": 38}, "explanation": map[string]any{"type": "string", "maxLength": 280}}}
}
func prompt(r game.ContentRequest) string {
	return fmt.Sprintf("Create one cosmetic variant. Base item: %q. Slot: %s. Rarity: %s. Practiced skill: %s. Challenge: %q. Preserve the skill relationship; do not alter stats, rarity, score, or correctness.", r.Item.Name, r.Item.Slot, r.Item.Rarity, r.Skill, r.ChallengeTitle)
}
func decodeContent(raw []byte) (game.GeneratedContent, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value wireContent
	if err := decoder.Decode(&value); err != nil {
		return game.GeneratedContent{}, fmt.Errorf("AI output violates schema: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return game.GeneratedContent{}, errors.New("AI output contains trailing JSON")
	}
	value.Name = cleanText(value.Name)
	value.Flavor = cleanText(value.Flavor)
	value.Sprite = sanitize(value.Sprite)
	value.Explanation = cleanText(value.Explanation)
	if value.Name == "" || value.Flavor == "" || value.Sprite == "" || value.Explanation == "" {
		return game.GeneratedContent{}, errors.New("AI output contains an empty field")
	}
	if utf8.RuneCountInString(value.Name) > 40 || utf8.RuneCountInString(value.Flavor) > 160 || utf8.RuneCountInString(value.Explanation) > 280 {
		return game.GeneratedContent{}, errors.New("AI output exceeds text limits")
	}
	lines := strings.Split(value.Sprite, "\n")
	if len(lines) > 3 {
		return game.GeneratedContent{}, errors.New("AI sprite exceeds three lines")
	}
	for _, line := range lines {
		if utf8.RuneCountInString(line) > 12 {
			return game.GeneratedContent{}, errors.New("AI sprite exceeds twelve columns")
		}
		for _, r := range line {
			if r < 0x20 || r > 0x7e {
				return game.GeneratedContent{}, errors.New("AI sprite must be printable ASCII")
			}
		}
	}
	return game.GeneratedContent{Name: value.Name, Flavor: value.Flavor, Sprite: value.Sprite, Explanation: value.Explanation}, nil
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(sanitize(value)), " ")
}

func sanitize(s string) string {
	var out strings.Builder
	state := 0 // normal, escape, CSI, OSC, OSC-after-escape
	for _, r := range s {
		switch state {
		case 1:
			switch r {
			case '[':
				state = 2
			case ']':
				state = 3
			default:
				state = 0
			}
			continue
		case 2:
			if r >= 0x40 && r <= 0x7e {
				state = 0
			}
			continue
		case 3:
			switch r {
			case '\a':
				state = 0
			case 0x1b:
				state = 4
			}
			continue
		case 4:
			if r == '\\' {
				state = 0
			} else {
				state = 3
			}
			continue
		}
		if r == 0x1b {
			state = 1
			continue
		}
		if r == 0x9b {
			state = 2
			continue
		}
		if r == 0x9d {
			state = 3
			continue
		}
		if r < 0x20 && r != '\n' && r != '\t' {
			continue
		}
		if r >= 0x7f && r <= 0x9f {
			continue
		}
		if r == 0x202e || r == 0x202d || r == 0x202a || r == 0x202b || r == 0x202c || r == 0x2066 || r == 0x2067 || r == 0x2068 || r == 0x2069 {
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
func safeRemoteError(raw []byte) string {
	s := sanitize(string(raw))
	if len(s) > 512 {
		s = s[:512]
	}
	return s
}
func cacheKey(model string, input game.ContentRequest) (string, error) {
	raw, err := json.Marshal(struct {
		Version, Model string
		Input          game.ContentRequest
	}{generatorVersion, model, input})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
func validateEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme == "https" && u.Hostname() == "openrouter.ai" && u.Path == "/api/v1/chat/completions" {
		return nil
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
	}
	return errors.New("OpenRouter endpoint must be the official HTTPS endpoint (HTTP loopback is allowed for tests)")
}
