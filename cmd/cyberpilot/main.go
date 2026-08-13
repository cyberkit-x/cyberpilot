package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cyberkit-x/cyberpilot/internal/buildinfo"
	"github.com/cyberkit-x/cyberpilot/internal/cli"
	"github.com/cyberkit-x/cyberpilot/internal/configuration"
	"github.com/cyberkit-x/cyberpilot/internal/credentials"
	"github.com/cyberkit-x/cyberpilot/internal/evidence/artifact"
	"github.com/cyberkit-x/cyberpilot/internal/model"
	modelprovider "github.com/cyberkit-x/cyberpilot/internal/model/openai"
	"github.com/cyberkit-x/cyberpilot/internal/platform"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
	"github.com/cyberkit-x/cyberpilot/internal/runner/oci"
	localruntime "github.com/cyberkit-x/cyberpilot/internal/runtime"
	"github.com/cyberkit-x/cyberpilot/internal/service"
	"github.com/cyberkit-x/cyberpilot/internal/skills"
	store "github.com/cyberkit-x/cyberpilot/internal/storage/sqlite"
	"github.com/cyberkit-x/cyberpilot/internal/tui"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		var exit *cli.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.Code)
		}
		os.Exit(3)
	}
}

func run(ctx context.Context, args []string) error {
	paths, err := platform.ResolvePaths()
	if err != nil {
		return err
	}
	configStore := configuration.FileStore{Path: paths.ConfigFile}
	daemonCommand := func(ctx context.Context, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("daemon mode accepts no arguments")
		}
		config, err := configStore.Load(ctx)
		if err != nil {
			return err
		}
		image := os.Getenv("CYBERPILOT_SANDBOX_IMAGE")
		if image == "" {
			return fmt.Errorf("CYBERPILOT_SANDBOX_IMAGE must name the pinned local sandbox image")
		}
		artifactStore, err := artifact.Open(paths.ArtifactsDir)
		if err != nil {
			return err
		}
		native := credentials.Native{}
		provider := &modelprovider.Provider{BaseURL: config.Model.BaseURL, Model: config.Model.Model, Credential: func(credentialCtx context.Context) (string, error) {
			return native.Get(credentialCtx, config.Model.CredentialRef)
		}}
		adapter := oci.New(config.Runner.Provider, config.Runner.Provider, config.Runner.Connection)
		index := &skills.Index{Sources: []skills.Source{skills.BundledSource{}}}
		factory := func(sessions *service.SessionService, _ *store.Store) localruntime.SessionWorker {
			return &localruntime.Worker{Sessions: sessions, Model: provider, Skills: index, Runner: &runner.Manager{Provider: adapter, WorkspaceRoot: filepath.Join(paths.DataDir, "workspaces")}, Artifacts: artifactStore, Image: image}
		}
		daemon, err := localruntime.NewDaemonWithWorker(paths, factory)
		if err != nil {
			return err
		}
		defer daemon.Close()
		daemonCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
		return daemon.Serve(daemonCtx)
	}
	configCommand := func(ctx context.Context, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("config accepts no arguments yet")
		}
		config, err := configStore.Load(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(configuration.Redact(config))
	}
	initCommand := func(ctx context.Context, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("init accepts no arguments")
		}
		native := credentials.Native{}
		discovery := runner.Discovery{}
		initializer := configuration.Initializer{Configs: configStore, Credentials: native,
			ProbeModel: func(ctx context.Context, profile configuration.ModelProfile, secret string) (report model.CapabilityReport, err error) {
				provider := modelprovider.Provider{BaseURL: profile.BaseURL, Model: profile.Model, Credential: func(context.Context) (string, error) { return secret, nil }}
				return provider.Probe(ctx)
			},
			ProbeRunner: func(ctx context.Context, profile configuration.RunnerProfile) error {
				image := os.Getenv("CYBERPILOT_SANDBOX_IMAGE")
				if image == "" {
					return fmt.Errorf("CYBERPILOT_SANDBOX_IMAGE must name an existing local probe image")
				}
				summaries, err := discovery.Discover(ctx)
				if err != nil {
					return err
				}
				selected, err := runner.SelectProvider(summaries, profile.Provider)
				if err != nil {
					return err
				}
				probeCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
				defer cancel()
				return runner.ProbeLifecycle(probeCtx, nil, selected, image)
			},
		}
		command := cli.InitCommand{Input: os.Stdin, Output: os.Stdout, Configs: configStore, Initializer: initializer, Discover: discovery.Discover}
		return command.Run(ctx)
	}
	tuiCommand := func(ctx context.Context, args []string) error {
		if _, err := configStore.Load(ctx); err != nil {
			return fmt.Errorf("CyberPilot is not initialized; run 'cyberpilot init': %w", err)
		}
		connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		client, err := localruntime.EnsureClient(connectCtx, paths, nil)
		if err != nil {
			return err
		}
		defer client.Close()
		model := tui.New(client)
		if err := model.Refresh(ctx); err != nil {
			return err
		}
		_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
		return err
	}
	execCommand := func(ctx context.Context, args []string) error {
		if _, err := configStore.Load(ctx); err != nil {
			return err
		}
		connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		client, err := localruntime.EnsureClient(connectCtx, paths, nil)
		if err != nil {
			return err
		}
		defer client.Close()
		return (cli.ExecCommand{Input: os.Stdin, Output: os.Stdout, Error: os.Stderr, Client: client, PollInterval: 200 * time.Millisecond}).Run(ctx, args)
	}
	app := cli.App{Input: os.Stdin, Output: os.Stdout, Error: os.Stderr, Init: initCommand, Config: configCommand, Exec: execCommand, TUI: tuiCommand, Daemon: daemonCommand, Version: buildinfo.String}
	return app.Run(ctx, args)
}
