package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecPromptSourcesAndOptions(t *testing.T) {
	file := filepath.Join(t.TempDir(), "prompt.txt")
	if err := os.WriteFile(file, []byte("from file"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		args        []string
		stdin, want string
	}{
		{"arg", []string{"--json", "--detach", "--model", "m", "--runner", "r", "--max-actions", "3", "prompt value"}, "", "prompt value"},
		{"stdin", []string{"-"}, "from stdin", "from stdin"},
		{"file", []string{"--prompt-file", file}, "", "from file"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseExecOptions(test.args, strings.NewReader(test.stdin))
			if err != nil || got.Prompt != test.want {
				t.Fatalf("got=%#v err=%v", got, err)
			}
			if strings.Contains(got.SafeSummary(), got.Prompt) {
				t.Fatal("prompt entered diagnostics")
			}
		})
	}
}

func TestExecInputValidation(t *testing.T) {
	for _, args := range [][]string{nil, {""}, {"a", "b"}, {"--prompt-file", "x", "prompt"}, {"--max-actions", "-1", "prompt"}} {
		if _, err := ParseExecOptions(args, strings.NewReader("")); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
