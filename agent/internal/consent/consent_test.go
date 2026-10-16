package consent

import (
	"testing"
	"time"
)

func TestNotAcceptedByDefault(t *testing.T) {
	if HasAccepted(t.TempDir(), PolicyVersion) {
		t.Fatal("fresh install should not count as accepted")
	}
}

func TestRecordAcceptanceIsRemembered(t *testing.T) {
	dir := t.TempDir()
	if err := RecordAcceptance(dir, PolicyVersion, "sara", time.Now()); err != nil {
		t.Fatal(err)
	}
	if !HasAccepted(dir, PolicyVersion) {
		t.Fatal("acceptance not remembered")
	}
}

func TestNewPolicyVersionNeedsNewAcceptance(t *testing.T) {
	dir := t.TempDir()
	if err := RecordAcceptance(dir, 1, "sara", time.Now()); err != nil {
		t.Fatal(err)
	}
	if HasAccepted(dir, 2) {
		t.Fatal("changed policy must be shown again")
	}
}

func TestPolicyTextStatesWhatIsNotCollected(t *testing.T) {
	for _, must := range []string{"never", "typed", "screen"} {
		if !contains(PolicyText, must) {
			t.Fatalf("policy text should mention %q", must)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
