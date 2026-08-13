//go:build darwin

package credentials

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

type Native struct{}

func (Native) Put(ctx context.Context, name, secret string) (string, error) {
	// With -w as the final argument security prompts for the password. Supplying
	// that prompt through stdin keeps the credential out of argv and process
	// inspection tools.
	if output, err := command(ctx, strings.NewReader(secret+"\n"), "security", "add-generic-password", "-U", "-s", "cyberpilot", "-a", name, "-w"); err != nil {
		return "", fmt.Errorf("store keychain credential: %w: %s", err, bytes.TrimSpace(output))
	}
	return "keychain:" + name, nil
}
func (Native) Get(ctx context.Context, ref string) (string, error) {
	name := strings.TrimPrefix(ref, "keychain:")
	output, err := command(ctx, nil, "security", "find-generic-password", "-s", "cyberpilot", "-a", name, "-w")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
func (Native) Delete(ctx context.Context, ref string) error {
	name := strings.TrimPrefix(ref, "keychain:")
	_, err := command(ctx, nil, "security", "delete-generic-password", "-s", "cyberpilot", "-a", name)
	return err
}
