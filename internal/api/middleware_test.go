package api

import (
	"regexp"
	"testing"
)

func TestGenerateRequestIDProducesUniqueWellFormedValues(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`^\d{14}-[0-9a-f]{16}$`)
	seen := make(map[string]struct{}, 128)

	for i := 0; i < 128; i++ {
		requestID := generateRequestID()
		if !pattern.MatchString(requestID) {
			t.Fatalf("generateRequestID() = %q, want timestamp-random format", requestID)
		}
		if _, exists := seen[requestID]; exists {
			t.Fatalf("generateRequestID() returned duplicate value %q", requestID)
		}
		seen[requestID] = struct{}{}
	}
}
