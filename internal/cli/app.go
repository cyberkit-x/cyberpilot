package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type CommandFunc func(context.Context, []string) error
type App struct {
	Input                           io.Reader
	Output, Error                   io.Writer
	Init, Config, Exec, TUI, Daemon CommandFunc
	Version                         func() string
}

func (a App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return call(a.TUI, ctx, nil, "terminal interface")
	}
	switch args[0] {
	case "version", "--version":
		if len(args) != 1 {
			return errors.New("version accepts no arguments")
		}
		_, err := fmt.Fprintln(a.Output, a.Version())
		return err
	case "init":
		return call(a.Init, ctx, args[1:], "init")
	case "config":
		return call(a.Config, ctx, args[1:], "config")
	case "exec":
		return call(a.Exec, ctx, args[1:], "exec")
	case "__daemon":
		return call(a.Daemon, ctx, args[1:], "daemon")
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func call(command CommandFunc, ctx context.Context, args []string, name string) error {
	if command == nil {
		return fmt.Errorf("%s is unavailable", name)
	}
	return command(ctx, args)
}
