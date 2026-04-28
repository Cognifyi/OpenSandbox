package controller

import (
	"testing"
	"time"
)

func TestParseBrowserLaunchOutput(t *testing.T) {
	t.Parallel()

	port, pid, dataDir, err := parseBrowserLaunchOutput("CDP_PORT:30123\nBROWSER_PID:456\nUSER_DATA_DIR:/tmp/browser-123\n")
	if err != nil {
		t.Fatalf("parseBrowserLaunchOutput returned error: %v", err)
	}
	if port != 30123 {
		t.Fatalf("unexpected port: got %d want %d", port, 30123)
	}
	if pid != 456 {
		t.Fatalf("unexpected pid: got %d want %d", pid, 456)
	}
	if dataDir != "/tmp/browser-123" {
		t.Fatalf("unexpected data dir: got %q", dataDir)
	}
}

func TestParseBrowserLaunchOutputRequiresMetadata(t *testing.T) {
	t.Parallel()

	if _, _, _, err := parseBrowserLaunchOutput("CDP_PORT:30123\n"); err == nil {
		t.Fatal("expected parseBrowserLaunchOutput to fail when metadata is incomplete")
	}
}

func TestBrowserSessionStoreListIncludesPersistedSessions(t *testing.T) {
	store := &browserSessionStore{
		browsers: map[string]*BrowserSession{
			"b": {
				PID:       2,
				Port:      30002,
				DataDir:   "/tmp/b",
				CreatedAt: time.Unix(2, 0),
			},
			"a": {
				PID:       1,
				Port:      30001,
				DataDir:   "/tmp/a",
				CreatedAt: time.Unix(1, 0),
			},
		},
	}

	sessions := store.list()
	if len(sessions) != 2 {
		t.Fatalf("unexpected session count: got %d want %d", len(sessions), 2)
	}
	if sessions[0]["sessionId"] != "a" {
		t.Fatalf("unexpected first session id: got %v want %q", sessions[0]["sessionId"], "a")
	}
	if sessions[1]["sessionId"] != "b" {
		t.Fatalf("unexpected second session id: got %v want %q", sessions[1]["sessionId"], "b")
	}
}
