package buildinfo

import "testing"

func TestString(t *testing.T) {
	got := String()
	want := "cyberpilot dev (commit unknown, built unknown)"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
