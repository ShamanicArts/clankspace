package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShamanicArts/clankspace/internal/client"
	"github.com/ShamanicArts/clankspace/internal/domain"
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

func TestRunMutationsFallBackAgainstPreJujutsuServer(t *testing.T) {
	requests := 0
	legacyBodies := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["vcs"]; ok {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid JSON: json: unknown field \"vcs\""}`))
			return
		}
		legacyBodies = append(legacyBodies, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"run":{"id":"run-1","projectId":"project-1","agentId":"agent-1","agentName":"agent","principalId":"principal-1","principalName":"Owner","role":"primary","runType":"interactive","startedAt":"2026-08-05T00:00:00Z"}}`))
	}))
	defer server.Close()
	c := client.New(server.URL, "token")
	if _, err := c.StartRun(t.Context(), domain.StartRunInput{ProjectID: "project-1", Branch: "main", VCS: "jj", JJWorkspace: "default", JJChangeID: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", JJCommitID: "1111111111111111111111111111111111111111"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.EndRun(t.Context(), "run-1", domain.EndRunInput{Outcome: "completed", DeliveryBranch: "agent/jj", VCS: "jj", DeliveryJJWorkspace: "default", DeliveryJJChangeID: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", DeliveryJJCommitID: "2222222222222222222222222222222222222222"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LinkRunDelivery(t.Context(), "run-1", domain.LinkRunDeliveryInput{DeliveryBranch: "agent/jj", VCS: "jj", DeliveryJJWorkspace: "default", DeliveryJJChangeID: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", DeliveryJJCommitID: "2222222222222222222222222222222222222222"}); err != nil {
		t.Fatal(err)
	}
	if requests != 6 || len(legacyBodies) != 3 {
		t.Fatalf("requests=%d legacy=%#v", requests, legacyBodies)
	}
	if legacyBodies[0]["branch"] != "main" || legacyBodies[1]["deliveryBranch"] != "agent/jj" || legacyBodies[2]["deliveryBranch"] != "agent/jj" {
		t.Fatalf("legacy Git provenance was not preserved: %#v", legacyBodies)
	}
	for _, body := range legacyBodies {
		if _, ok := body["jjChangeId"]; ok {
			t.Fatalf("legacy payload retained JJ fields: %#v", body)
		}
		if _, ok := body["deliveryJjChangeId"]; ok {
			t.Fatalf("legacy payload retained delivery JJ fields: %#v", body)
		}
	}
}
