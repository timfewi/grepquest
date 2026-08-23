package sui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGraphQLClientPaginatesAndParsesEvents(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		var request struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Variables["eventType"] != "0x1::social::ScoreSubmitted" {
			t.Fatalf("unexpected event filter: %#v", request.Variables)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			_, _ = w.Write([]byte(`{"data":{"events":{"nodes":[{"sender":{"address":"0xa"},"timestamp":"2026-08-22T10:00:00Z","contents":{"json":{"player":"0xa","season_id":[99,111,114,101],"score":"900","level":"4","receipt_sha256":[170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170,170],"proof_tier":1,"accuracy":95,"recall":80,"no_hint_clears":"7","elapsed_ms":"1200"}}}],"pageInfo":{"hasNextPage":true,"endCursor":"next"}}}}`))
			return
		}
		if request.Variables["after"] != "next" {
			t.Fatalf("missing pagination cursor: %#v", request.Variables)
		}
		_, _ = w.Write([]byte(`{"data":{"events":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}`))
	}))
	defer server.Close()

	client, err := NewGraphQLClient(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	events, err := client.FetchScoreEvents(context.Background(), "0x1::social::ScoreSubmitted", 2)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(events) != 1 {
		t.Fatalf("got %d requests and %d events", requests, len(events))
	}
	if events[0].SeasonID != "core" || events[0].ProofTier != Verified || events[0].Score != 900 || events[0].ReceiptSHA256 != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected event: %#v", events[0])
	}
}

func TestGraphQLClientRejectsUnsafeEndpointsAndRedirects(t *testing.T) {
	t.Parallel()
	if _, err := NewGraphQLClient("http://example.com/graphql", nil); err == nil {
		t.Fatal("expected cleartext remote endpoint to be rejected")
	}
	if _, err := NewGraphQLClient("https://user:secret@example.com/graphql", nil); err == nil {
		t.Fatal("expected endpoint credentials to be rejected")
	}
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("redirect target must not be contacted")
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusFound))
	defer redirect.Close()
	client, err := NewGraphQLClient(redirect.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.FetchScoreEvents(context.Background(), "0x1::social::ScoreSubmitted", 1); err == nil {
		t.Fatal("expected redirect response to fail")
	}
}

func TestGraphQLClientRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	client, err := NewGraphQLClient("https://graphql.testnet.sui.io/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.FetchScoreEvents(context.Background(), "0x1::other::Event", 1); err == nil {
		t.Fatal("expected wrong event type to fail before network access")
	}
	if _, err = client.FetchScoreEvents(context.Background(), "0x1::social::ScoreSubmitted", 21); err == nil {
		t.Fatal("expected excessive pagination to fail before network access")
	}
}

func TestGraphQLClientRejectsUnsafeEventText(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"events":{"nodes":[{"sender":{"address":"0xa"},"contents":{"json":{"player":"0xa","season_id":"bad\u001b[31m","score":1,"level":1,"receipt_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","proof_tier":0,"accuracy":1,"recall":1,"no_hint_clears":1,"elapsed_ms":1}}}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}`))
	}))
	defer server.Close()
	client, err := NewGraphQLClient(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.FetchScoreEvents(context.Background(), "0x1::social::ScoreSubmitted", 1); err == nil {
		t.Fatal("expected terminal control text to be rejected")
	}
}
