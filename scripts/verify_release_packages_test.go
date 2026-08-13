package scripts_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyReleasePackagesAcceptsCompleteAssetsAndRejectsCorruption(t *testing.T) {
	dist := t.TempDir()
	assets := []string{
		"cyberpilot_v0.1.0_linux_amd64.tar.gz",
		"cyberpilot_v0.1.0_linux_arm64.tar.gz",
		"cyberpilot_v0.1.0_darwin_amd64.tar.gz",
		"cyberpilot_v0.1.0_darwin_arm64.tar.gz",
		"cyberpilot_v0.1.0_windows_amd64.zip",
		"cyberpilot-release.spdx.json",
	}
	var checksums strings.Builder
	for _, name := range assets {
		data := []byte("fixture:" + name)
		if err := os.WriteFile(filepath.Join(dist, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		fmt.Fprintf(&checksums, "%x  %s\n", digest, name)
	}
	if err := os.WriteFile(filepath.Join(dist, "checksums.txt"), []byte(checksums.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func() error {
		command := exec.Command("sh", "verify-release-packages.sh", dist)
		command.Dir = "."
		command.Env = append(os.Environ(), "EXPECTED_VERSION=v0.1.0")
		return command.Run()
	}
	if err := run(); err != nil {
		t.Fatalf("complete release assets rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dist, assets[0]), []byte("corrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("corrupted release package passed checksum validation")
	}
}

func TestVerifyReleasePackagesRejectsMissingPlatform(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "checksums.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "cyberpilot-release.spdx.json"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", "verify-release-packages.sh", dist)
	command.Dir = "."
	command.Env = append(os.Environ(), "EXPECTED_VERSION=v0.1.0")
	if err := command.Run(); err == nil {
		t.Fatal("missing platform packages were accepted")
	}
}

func TestVerifyReleasePackagesRejectsMalformedTag(t *testing.T) {
	command := exec.Command("sh", "verify-release-packages.sh", t.TempDir())
	command.Dir = "."
	command.Env = append(os.Environ(), "EXPECTED_VERSION=v0.1.0-extra")
	if err := command.Run(); err == nil {
		t.Fatal("malformed release tag was accepted")
	}
}
