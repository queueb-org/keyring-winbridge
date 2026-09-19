package credential

import "testing"

func TestTargetNameMatchesNativeKeyringMapping(t *testing.T) {
	if got, want := targetName("example", "api-token"), "example:api-token"; got != want {
		t.Fatalf("targetName() = %q, want %q", got, want)
	}
}
