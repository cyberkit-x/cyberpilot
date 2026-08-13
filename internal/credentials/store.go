package credentials

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Store interface {
	Put(context.Context, string, string) (string, error)
	Get(context.Context, string) (string, error)
	Delete(context.Context, string) error
}

// command runs an operating-system credential helper. Keeping the invocation
// behind this small boundary lets tests prove that secret values are supplied
// over stdin and never become process arguments.
var command = func(ctx context.Context, stdin io.Reader, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	return cmd.CombinedOutput()
}

type Environment struct{}

func (Environment) Put(_ context.Context, name, _ string) (string, error) {
	return "", fmt.Errorf("environment credentials cannot be persisted; set %s", name)
}
func (Environment) Get(_ context.Context, ref string) (string, error) {
	name := strings.TrimPrefix(ref, "env:")
	if name == ref || name == "" {
		return "", fmt.Errorf("invalid environment credential reference")
	}
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", fmt.Errorf("credential environment variable %s is not set", name)
	}
	return value, nil
}
func (Environment) Delete(context.Context, string) error { return nil }
