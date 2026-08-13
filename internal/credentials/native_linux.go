//go:build linux

package credentials

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

type Native struct{ Environment }

func (Native) Put(ctx context.Context, name, secret string) (string, error) {
	if output, err := command(ctx, strings.NewReader(secret), "secret-tool", "store", "--label=CyberPilot model credential", "service", "cyberpilot", "account", name); err != nil {
		return "", fmt.Errorf("store Secret Service credential: %w: %s", err, bytes.TrimSpace(output))
	}
	return "secret-service:" + name, nil
}
func (Native) Get(ctx context.Context, ref string) (string, error) {
	name := strings.TrimPrefix(ref, "secret-service:")
	output, err := command(ctx, nil, "secret-tool", "lookup", "service", "cyberpilot", "account", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
func (Native) Delete(ctx context.Context, ref string) error {
	name := strings.TrimPrefix(ref, "secret-service:")
	_, err := command(ctx, nil, "secret-tool", "clear", "service", "cyberpilot", "account", name)
	return err
}
