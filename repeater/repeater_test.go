package repeater_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/secban/secban-agent/entry"
	"gitlab.com/secban/secban-agent/ipc"
	"gitlab.com/secban/secban-agent/offender"
	"gitlab.com/secban/secban-repeater/repeater"
)

func TestRepeaterBanSuccess(t *testing.T) {
	t.Parallel()

	var (
		receivedCount int32
		lastPath      string
		lastQuery     string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)

		lastPath = r.URL.Path
		lastQuery = r.URL.RawQuery

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"banned"}`))
	}))
	defer server.Close()

	rep := repeater.New([]string{server.URL}, 2*time.Second, nil)

	msg := ipc.Message{
		IP:   "198.51.100.5",
		Rule: "ssh",
		Offender: offender.Offender{
			Count: 3,
			Entries: []entry.Entry{
				{
					Message: "repeated auth failure",
				},
			},
		},
	}

	errs := rep.Repeat(context.Background(), msg)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got: %v", errs)
	}

	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Fatalf("expected 1 call, got %d", receivedCount)
	}

	expectedPath := "/ban/198.51.100.5/ssh"
	if lastPath != expectedPath {
		t.Fatalf("expected path %s, got %s", expectedPath, lastPath)
	}

	hostname, hostErr := os.Hostname()
	if hostErr != nil {
		hostname = "unknown"
	}

	expectedQuery := url.Values{"message": []string{"[repeater] " + hostname + " repeated auth failure"}}.Encode()
	if lastQuery != expectedQuery {
		t.Fatalf("expected query %s, got %s", expectedQuery, lastQuery)
	}
}

func TestRepeaterBanServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid rule"))
	}))
	defer server.Close()

	rep := repeater.New([]string{server.URL}, 2*time.Second, nil)

	msg := ipc.Message{
		IP:   "198.51.100.5",
		Rule: "badrule",
	}

	errs := rep.Repeat(context.Background(), msg)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestRepeaterSkipRepeatedBan(t *testing.T) {
	t.Parallel()

	var receivedCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&receivedCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rep := repeater.New([]string{server.URL}, 2*time.Second, nil)

	msg := ipc.Message{
		IP:   "198.51.100.5",
		Rule: "ssh",
		Offender: offender.Offender{
			Count: 1,
			Entries: []entry.Entry{
				{
					Message: "[repeater] remote-host repeated auth failure",
				},
			},
		},
	}

	errs := rep.Repeat(context.Background(), msg)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got: %v", errs)
	}

	if atomic.LoadInt32(&receivedCount) != 0 {
		t.Fatalf("expected 0 calls when message contains [repeater], got %d", receivedCount)
	}
}

func TestClientBanSkipRepeatedBan(t *testing.T) {
	t.Parallel()

	var receivedCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&receivedCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := repeater.NewClient(server.URL, 2*time.Second, nil)

	msg := ipc.Message{
		IP:   "198.51.100.5",
		Rule: "ssh",
		Offender: offender.Offender{
			Count: 2,
			Entries: []entry.Entry{
				{
					Message: "some initial error",
				},
				{
					Message: "[REPEATER] host2 auth failure",
				},
			},
		},
	}

	banErr := client.Ban(context.Background(), msg)
	if banErr != nil {
		t.Fatalf("expected nil error, got: %v", banErr)
	}

	if atomic.LoadInt32(&receivedCount) != 0 {
		t.Fatalf("expected 0 calls when message contains [repeater], got %d", receivedCount)
	}
}

func TestRepeaterRulesMap(t *testing.T) {
	t.Parallel()

	var (
		receivedCount int32
		lastPath      string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)

		lastPath = r.URL.Path

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"banned"}`))
	}))
	defer server.Close()

	rulesMap := map[string]string{
		"long_ban": "default",
	}

	rep := repeater.New([]string{server.URL}, 2*time.Second, rulesMap)

	msg := ipc.Message{
		IP:   "198.51.100.99",
		Rule: "long_ban",
	}

	errs := rep.Repeat(context.Background(), msg)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got: %v", errs)
	}

	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Fatalf("expected 1 call, got %d", receivedCount)
	}

	expectedPath := "/ban/198.51.100.99/default"
	if lastPath != expectedPath {
		t.Fatalf("expected path %s, got %s", expectedPath, lastPath)
	}
}

func TestRepeaterUnmappedRuleFallsBackToLocal(t *testing.T) {
	t.Parallel()

	var lastPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"banned"}`))
	}))
	defer server.Close()

	rulesMap := map[string]string{
		"long_ban": "default",
	}

	rep := repeater.New([]string{server.URL}, 2*time.Second, rulesMap)

	msg := ipc.Message{
		IP:   "198.51.100.101",
		Rule: "unmapped_rule",
	}

	errs := rep.Repeat(context.Background(), msg)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got: %v", errs)
	}

	expectedPath := "/ban/198.51.100.101/unmapped_rule"
	if lastPath != expectedPath {
		t.Fatalf("expected path %s, got %s", expectedPath, lastPath)
	}
}
