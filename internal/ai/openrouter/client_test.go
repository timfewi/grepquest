package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/timfewi/grepquest/internal/game"
)

type memoryCache struct {
	mu       sync.Mutex
	values   map[string][]byte
	requests int
}

func (c *memoryCache) GetAI(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, ok := c.values[key]
	return raw, ok, nil
}
func (c *memoryCache) PutAI(_ context.Context, key string, raw []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.values == nil {
		c.values = map[string][]byte{}
	}
	c.values[key] = raw
	return nil
}
func (c *memoryCache) AIRequestsToday(context.Context, time.Time) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requests, nil
}
func (c *memoryCache) RecordAIRequest(context.Context, time.Time, float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests++
	return nil
}

func TestGenerateEnforcesPrivacySchemaAndCache(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("authorization absent")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		provider := payload["provider"].(map[string]any)
		if provider["zdr"] != true || provider["data_collection"] != "deny" || provider["require_parameters"] != true {
			t.Errorf("unsafe provider policy: %#v", provider)
		}
		format := payload["response_format"].(map[string]any)
		schema := format["json_schema"].(map[string]any)
		if schema["strict"] != true {
			t.Error("schema is not strict")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"name\":\"Signal Crown\",\"flavor\":\"Finds quiet truths.\",\"sprite\":\"[S]\",\"explanation\":\"A cosmetic reward for literal matching.\"}"}}],"usage":{"cost":0.001}}`))
	}))
	defer server.Close()
	cache := &memoryCache{values: map[string][]byte{}}
	client, err := New(Config{APIKey: "secret", Endpoint: server.URL, Cache: cache})
	if err != nil {
		t.Fatal(err)
	}
	input := game.ContentRequest{Item: game.BuiltinItems[0], ChallengeTitle: "First Signal", Skill: "literal"}
	first, err := client.Generate(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Generate(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Signal Crown" || second.Source != "openrouter-cache" || calls != 1 {
		t.Fatalf("unexpected results: %#v %#v calls=%d", first, second, calls)
	}
}

func TestDisabledNeverCallsNetwork(t *testing.T) {
	client, err := New(Config{HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("network called"); return nil, nil })}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Generate(context.Background(), game.ContentRequest{})
	if err != ErrDisabled {
		t.Fatalf("got %v", err)
	}
}
func TestSanitizesControlAndRejectsInvalidShape(t *testing.T) {
	content, err := decodeContent([]byte(`{"name":"bad\u001b[31mred","flavor":"ok","sprite":"[x]","explanation":"ok"}`))
	if err != nil {
		t.Fatal(err)
	}
	if content.Name != "badred" {
		t.Fatalf("ANSI sequence survived: %q", content.Name)
	}
	for _, raw := range []string{`{"name":"ok","flavor":"ok","sprite":"1234567890123","explanation":"ok"}`, `{"name":"ok","flavor":"ok","sprite":"[x]","explanation":"ok","score":999}`} {
		if _, err := decodeContent([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
func TestEndpointAllowlist(t *testing.T) {
	for _, endpoint := range []string{"http://openrouter.ai/api/v1/chat/completions", "https://example.com/api/v1/chat/completions", "https://openrouter.ai/evil"} {
		if _, err := New(Config{Endpoint: endpoint}); err == nil {
			t.Errorf("accepted %s", endpoint)
		}
	}
}
func TestDailyBudget(t *testing.T) {
	cache := &memoryCache{values: map[string][]byte{}, requests: 2}
	client, err := New(Config{APIKey: "x", Cache: cache, MaxDailyRequests: 2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Generate(context.Background(), game.ContentRequest{Item: game.BuiltinItems[0]})
	if err == nil || !strings.Contains(err.Error(), "daily") {
		t.Fatalf("got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
