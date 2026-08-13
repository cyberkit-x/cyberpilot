package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/cyberkit-x/cyberpilot/internal/configuration"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
	"golang.org/x/term"
)

type InitCommand struct {
	Input       io.Reader
	Output      io.Writer
	Configs     configuration.Store
	Initializer configuration.Initializer
	Discover    func(context.Context) ([]runner.ProviderSummary, error)
}

func (command InitCommand) Run(ctx context.Context) error {
	reader := bufio.NewReader(command.Input)
	if current, err := command.Configs.Load(ctx); err == nil {
		redacted := configuration.Redact(current)
		fmt.Fprintf(command.Output, "Current model: %s / %s (%s)\n", redacted.Model.Provider, redacted.Model.Model, redacted.Model.BaseURL)
		fmt.Fprintf(command.Output, "Current runner: %s (%s)\n", redacted.Runner.Provider, redacted.Runner.Connection)
		answer, err := prompt(reader, command.Output, "Replace the current configuration? [y/N]: ", false)
		if err != nil {
			return err
		}
		if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
			fmt.Fprintln(command.Output, "Configuration unchanged.")
			return nil
		}
	} else if !errors.Is(err, configuration.ErrNotConfigured) {
		return err
	}

	baseURL, err := prompt(reader, command.Output, "OpenAI-compatible base URL: ", true)
	if err != nil {
		return err
	}
	modelName, err := prompt(reader, command.Output, "Model name: ", true)
	if err != nil {
		return err
	}
	credential, err := promptSecret(reader, command.Input, command.Output, "API key: ")
	if err != nil {
		return err
	}
	summaries, err := command.Discover(ctx)
	if err != nil {
		return err
	}
	for index, summary := range summaries {
		mode := "rootful"
		if summary.Rootless {
			mode = "rootless"
		}
		fmt.Fprintf(command.Output, "  %d. %s %s (%s, %s)\n", index+1, summary.Provider, summary.Version, summary.Endpoint, mode)
	}
	selected := ""
	if len(summaries) > 1 {
		value, err := prompt(reader, command.Output, "Select runner [1-"+strconv.Itoa(len(summaries))+"]: ", true)
		if err != nil {
			return err
		}
		index, err := strconv.Atoi(value)
		if err != nil || index < 1 || index > len(summaries) {
			return errors.New("runner selection is invalid")
		}
		selected = summaries[index-1].Provider
	}
	provider, err := runner.SelectProvider(summaries, selected)
	if err != nil {
		return err
	}
	candidate := configuration.Candidate{
		Model:      configuration.ModelProfile{Provider: "openai-compatible", Model: modelName, BaseURL: baseURL},
		Runner:     configuration.RunnerProfile{Provider: provider.Provider, Connection: provider.Endpoint, Rootless: provider.Rootless},
		Credential: credential,
	}
	if _, err := command.Initializer.Initialize(ctx, candidate); err != nil {
		return err
	}
	fmt.Fprintln(command.Output, "CyberPilot is ready.")
	return nil
}

func promptSecret(reader *bufio.Reader, input io.Reader, output io.Writer, label string) (string, error) {
	if _, err := io.WriteString(output, label); err != nil {
		return "", err
	}
	if file, ok := input.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		data, err := term.ReadPassword(int(file.Fd()))
		_, _ = io.WriteString(output, "\n")
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", errors.New("a required value was empty")
		}
		return value, nil
	}
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("a required value was empty")
	}
	return value, nil
}

func prompt(reader *bufio.Reader, output io.Writer, label string, required bool) (string, error) {
	if _, err := io.WriteString(output, label); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	value = strings.TrimSpace(value)
	if required && value == "" {
		return "", errors.New("a required value was empty")
	}
	return value, nil
}
