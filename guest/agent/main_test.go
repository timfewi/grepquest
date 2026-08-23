package main

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
)

func TestReadyRequest(t *testing.T) {
	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		handle(server)
		close(done)
	}()

	if err := json.NewEncoder(client).Encode(request{Type: "ready"}); err != nil {
		t.Fatal(err)
	}
	var response ack
	if err := json.NewDecoder(client).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Error != "" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	<-done
}

func TestFDConnNormalizesFailedSyscallCounts(t *testing.T) {
	conn := &fdConn{fd: -1}
	if n, err := conn.Read(make([]byte, 1)); n != 0 || err == nil {
		t.Fatalf("unexpected read result: n=%d err=%v", n, err)
	}
	if n, err := conn.Write([]byte("x")); n != 0 || err == nil {
		t.Fatalf("unexpected write result: n=%d err=%v", n, err)
	}
}

func TestSanitizeErrorRemovesTerminalControls(t *testing.T) {
	got := sanitizeError(fakeError("bad\x1b[31m\x07value"))
	if strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("control characters survived: %q", got)
	}
}
func TestHasParent(t *testing.T) {
	if !hasParent("a/../b") {
		t.Fatal("parent segment accepted")
	}
	if hasParent("a..b/file") {
		t.Fatal("safe filename rejected")
	}
}

type fakeError string

func (e fakeError) Error() string { return string(e) }
