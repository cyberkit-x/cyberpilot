package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/scheduler"
)

const MaxPromptBytes = 1 << 20

type ExecOptions struct {
	Prompt                      string
	JSON, Detach                bool
	ModelProfile, RunnerProfile string
	Budget                      scheduler.Budget
}

func ParseExecOptions(args []string, stdin io.Reader) (ExecOptions, error) {
	flags := flag.NewFlagSet("exec", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options ExecOptions
	var promptFile string
	var timeout time.Duration
	flags.BoolVar(&options.JSON, "json", false, "")
	flags.BoolVar(&options.Detach, "detach", false, "")
	flags.StringVar(&options.ModelProfile, "model", "", "")
	flags.StringVar(&options.RunnerProfile, "runner", "", "")
	flags.StringVar(&promptFile, "prompt-file", "", "")
	flags.DurationVar(&timeout, "timeout", 0, "")
	flags.IntVar(&options.Budget.MaxActions, "max-actions", 0, "")
	flags.Int64Var(&options.Budget.MaxInputTokens, "max-input-tokens", 0, "")
	flags.Int64Var(&options.Budget.MaxOutputTokens, "max-output-tokens", 0, "")
	flags.Float64Var(&options.Budget.MaxCost, "max-cost", 0, "")
	if err := flags.Parse(args); err != nil {
		return options, fmt.Errorf("invalid exec options: %w", err)
	}
	positional := flags.Args()
	sources := 0
	if len(positional) == 1 {
		sources++
	}
	if len(positional) > 1 {
		return options, errors.New("exec accepts one prompt")
	}
	if promptFile != "" {
		sources++
	}
	if sources != 1 {
		return options, errors.New("provide exactly one prompt source")
	}
	var data []byte
	var err error
	switch {
	case promptFile != "":
		data, err = os.ReadFile(promptFile)
	case positional[0] == "-":
		data, err = io.ReadAll(io.LimitReader(stdin, MaxPromptBytes+1))
	default:
		data = []byte(positional[0])
	}
	if err != nil {
		return options, fmt.Errorf("read prompt: %w", err)
	}
	if len(data) > MaxPromptBytes {
		return options, errors.New("prompt exceeds 1 MiB limit")
	}
	options.Prompt = strings.TrimSpace(string(data))
	if options.Prompt == "" {
		return options, errors.New("prompt is empty")
	}
	if timeout < 0 || options.Budget.MaxActions < 0 || options.Budget.MaxInputTokens < 0 || options.Budget.MaxOutputTokens < 0 || options.Budget.MaxCost < 0 {
		return options, errors.New("budgets cannot be negative")
	}
	if timeout > 0 {
		options.Budget.Deadline = time.Now().Add(timeout)
	}
	return options, nil
}

func (o ExecOptions) SafeSummary() string {
	return fmt.Sprintf("json=%t detach=%t model=%q runner=%q prompt_bytes=%d", o.JSON, o.Detach, o.ModelProfile, o.RunnerProfile, len(o.Prompt))
}
