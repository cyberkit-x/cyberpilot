//go:build darwin || linux

package credentials

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestNativePutNeverPlacesSecretInCommandArguments(t *testing.T) {
	const secret = "credential-that-must-not-enter-argv"
	original := command
	t.Cleanup(func() { command = original })
	command = func(_ context.Context, stdin io.Reader, executable string, args ...string) ([]byte, error) {
		if strings.Contains(executable+" "+strings.Join(args, " "), secret) {
			t.Fatal("secret entered command arguments")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), secret) {
			t.Fatal("secret was not supplied over stdin")
		}
		return nil, nil
	}
	ref, err := (Native{}).Put(context.Background(), "default", secret)
	if err != nil {
		t.Fatal(err)
	}
	if ref == "" || strings.Contains(ref, secret) {
		t.Fatalf("unsafe credential reference %q", ref)
	}
}
