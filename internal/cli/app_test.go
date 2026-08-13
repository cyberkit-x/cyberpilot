package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCommandTree(t *testing.T) {
	commands := []string{}
	record := func(name string) CommandFunc {
		return func(_ context.Context, args []string) error {
			commands = append(commands, name+":"+strings.Join(args, ","))
			return nil
		}
	}
	var output bytes.Buffer
	app := App{Output: &output, Init: record("init"), Config: record("config"), Exec: record("exec"), TUI: record("tui"), Daemon: record("daemon"), Version: func() string { return "v1" }}
	for _, args := range [][]string{nil, {"init"}, {"config"}, {"exec", "prompt"}, {"__daemon"}, {"version"}} {
		if err := app.Run(context.Background(), args); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(commands, "|") != "tui:|init:|config:|exec:prompt|daemon:" || strings.TrimSpace(output.String()) != "v1" {
		t.Fatalf("commands=%v output=%q", commands, output.String())
	}
	if err := app.Run(context.Background(), []string{"unknown"}); err == nil {
		t.Fatal("unknown command accepted")
	}
}
