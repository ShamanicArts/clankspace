package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestRequestFingerprintCannotBeRotatedWithUserAgent(t *testing.T) {
	first := httptest.NewRequest("POST", "/api/v1/auth/password", nil)
	first.RemoteAddr = "192.0.2.10:1234"
	first.Header.Set("User-Agent", "first-agent")
	second := httptest.NewRequest("POST", "/api/v1/auth/password", nil)
	second.RemoteAddr = "192.0.2.10:9876"
	second.Header.Set("User-Agent", "rotated-agent")
	if requestFingerprint(first) != requestFingerprint(second) {
		t.Fatal("user-agent rotation bypassed the source rate-limit key")
	}
}
