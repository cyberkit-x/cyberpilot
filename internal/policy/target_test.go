package policy

import (
	"testing"
)

func TestCanonicalTargets(t *testing.T) {
	tests := map[string]string{"HTTPS://Example.COM/api#x": "https://example.com:443/api", "api.example.com": "api.example.com", "api.example.com:8443": "api.example.com:8443", "192.0.2.1": "192.0.2.1", "[2001:db8::1]:443": "[2001:db8::1]:443", "192.0.2.99/24": "192.0.2.0/24", "*.Example.COM": "*.example.com"}
	for input, want := range tests {
		got, err := ParseTarget(input)
		if err != nil || got.Canonical != want {
			t.Errorf("%q => %#v err=%v want=%q", input, got, err, want)
		}
	}
}
func TestAmbiguousTargetsRejected(t *testing.T) {
	for _, value := range []string{"example", "*example.com", "ftp://example.com", "user:pass@example.com", "example.com:0", "10.0.0.999", ""} {
		if _, err := ParseTarget(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}
func TestScopeConfirmationKeepsExplicitForms(t *testing.T) {
	wildcard, _ := ParseTarget("*.example.com")
	root, _ := ParseTarget("example.com")
	if wildcard.Canonical == root.Canonical || wildcard.Kind == root.Kind {
		t.Fatal("wildcard collapsed into root host")
	}
}
