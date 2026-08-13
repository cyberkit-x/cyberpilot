package credentials

import (
	"context"
	"os"
	"testing"
)

func TestEnvironmentReference(t *testing.T) {
	t.Setenv("CYBERPILOT_TEST_SECRET", "value")
	got, err := (Environment{}).Get(context.Background(), "env:CYBERPILOT_TEST_SECRET")
	if err != nil {
		t.Fatal(err)
	}
	if got != "value" {
		t.Fatalf("got %q", got)
	}
	if _, err := (Environment{}).Get(context.Background(), "value"); err == nil {
		t.Fatal("expected invalid reference")
	}
	_ = os.Unsetenv("CYBERPILOT_TEST_SECRET")
}
