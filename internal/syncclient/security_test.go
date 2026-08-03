package syncclient

import "testing"

func TestReplicaTransportRequiresHTTPSOutsidePrivateNetworks(t *testing.T) {
	allowed := []string{"https://clank.example.com", "http://127.0.0.1:8080", "http://localhost:8080", "http://10.1.2.3:8080", "http://100.64.12.4:8080"}
	for _, candidate := range allowed {
		if err := validateRemoteURL(candidate); err != nil {
			t.Fatalf("expected %s to be allowed: %v", candidate, err)
		}
	}
	blocked := []string{"http://8.8.8.8:8080", "ftp://127.0.0.1:8080", "not a url"}
	for _, candidate := range blocked {
		if err := validateRemoteURL(candidate); err == nil {
			t.Fatalf("expected %s to be rejected", candidate)
		}
	}
}
