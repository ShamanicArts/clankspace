package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/client"
)

func TestDoWithKeySendsExactIdempotencyKey(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	c := client.New(server.URL, "token")
	if err := c.DoWithKey(context.Background(), http.MethodPost, "/test", "scenario:record-1", map[string]bool{"ok": true}, &map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if got != "scenario:record-1" {
		t.Fatalf("idempotency key = %q", got)
	}
}

func TestDoWithKeyRejectsEmptyMutationKey(t *testing.T) {
	c := client.New("https://example.invalid", "token")
	err := c.DoWithKey(context.Background(), http.MethodPost, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("expected empty mutation key to fail before making a request")
	}
}
