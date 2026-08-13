package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestExecGoldenOutcomes(t *testing.T) {
	tests := []struct {
		name  string
		state string
		code  int
	}{{"no finding", "completed", 0}, {"finding", "completed", 1}, {"approval", "needs-input", 2}, {"runner", "failed", 3}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ExecResult{State: domain.SessionState(test.state)}
			if test.name == "finding" {
				result.VerifiedFindings = 1
			}
			var output bytes.Buffer
			if err := json.NewEncoder(&output).Encode(result); err != nil {
				t.Fatal(err)
			}
			var decoded ExecResult
			if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
				t.Fatal(err)
			}
			code := 0
			if decoded.VerifiedFindings > 0 {
				code = 1
			}
			if decoded.State == domain.SessionNeedsInput || decoded.State == domain.SessionBlocked {
				code = 2
			}
			if decoded.State == domain.SessionFailed {
				code = 3
			}
			if code != test.code {
				t.Fatalf("code=%d want=%d", code, test.code)
			}
		})
	}
}

func TestExecInvalidInputExitFour(t *testing.T) {
	command := ExecCommand{Input: &bytes.Buffer{}, Output: &bytes.Buffer{}, Error: &bytes.Buffer{}, Client: &execClientFake{}}
	err := command.Run(context.Background(), []string{"invalid scope"})
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 4 {
		t.Fatalf("err=%v", err)
	}
}
