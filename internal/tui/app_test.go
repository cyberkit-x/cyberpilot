package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"strings"
	"testing"
	"time"
)

func fixtures() []domain.Session {
	return []domain.Session{{ID: domain.MustNewID(), Name: "Authorization review with a very long descriptive title", Objective: "Assess Unicode API 目标 and authorization behavior without horizontal scrolling", Targets: []string{"https://fixture.local/api/objects/very/long/path"}, Goals: []string{"IDOR", "JWT"}, State: domain.SessionNeedsInput}, {ID: domain.MustNewID(), Name: "JWT assessment", Objective: "Assess JWT", Targets: []string{"https://fixture.local"}, Goals: []string{"auth"}, State: domain.SessionRunning}}
}
func TestOverviewListsAndResponsiveWidths(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {80, 24}, {140, 40}} {
		m := New(structClient{})
		m.Sessions = fixtures()
		m.Width, m.Height = size[0], size[1]
		view := m.View()
		for _, label := range []string{"NEEDS INPUT", "OTHER SESSIONS"} {
			if !strings.Contains(view, label) {
				t.Fatalf("size=%v missing %q in %q", size, label, view)
			}
		}
		for _, line := range strings.Split(view, "\n") {
			if len([]rune(line)) > size[0] {
				t.Fatalf("size=%v line too wide: %q", size, line)
			}
		}
	}
}
func TestKeyboardFlowAndSessionDetail(t *testing.T) {
	m := New(structClient{})
	m.Sessions = fixtures()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	detail := updated.(Model)
	if detail.Screen != SessionDetail || !strings.Contains(detail.View(), "Objective") || !strings.Contains(detail.View(), "COVERAGE GAPS") {
		t.Fatalf("detail=%q", detail.View())
	}
	updated, _ = detail.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if updated.(Model).Screen != InstructionEditor {
		t.Fatal("instruction flow not opened")
	}
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.(Model).Screen != Overview {
		t.Fatal("did not return overview")
	}
}
func TestReconnectAndSearchMultipleSessions(t *testing.T) {
	m := New(nil)
	if !strings.Contains(m.View(), "RECONNECTING") {
		t.Fatal("reconnect state missing")
	}
	m = New(structClient{})
	m.Sessions = fixtures()
	m.Query = "JWT"
	if len(m.visible()) != 1 {
		t.Fatalf("visible=%v", m.visible())
	}
}

func TestOverviewCreateAndSearchKeys(t *testing.T) {
	m := New(structClient{})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if updated.(Model).Screen != CreateEditor {
		t.Fatal("new-session editor did not open")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if updated.(Model).Screen != SearchEditor {
		t.Fatal("search editor did not open")
	}
}

type structClient struct{}

func (structClient) Call(_ context.Context, _ string, _ any, output any) error {
	switch value := output.(type) {
	case *[]domain.Session:
		*value = fixtures()
	case *domain.Session:
		*value = fixtures()[0]
	case *[]domain.Event:
		event, _ := domain.NewEvent(fixtures()[0].ID, 1, "session.created", time.Now(), map[string]any{})
		*value = []domain.Event{event}
	}
	return nil
}

func TestInterventionRPCFlows(t *testing.T) {
	m := New(structClient{})
	m.Current = fixtures()[0]
	if err := m.Refresh(context.Background()); err != nil || len(m.Sessions) != 2 {
		t.Fatalf("sessions=%v err=%v", m.Sessions, err)
	}
	if err := m.UpdateScope(context.Background(), []string{"https://fixture.local"}, false); err == nil {
		t.Fatal("unconfirmed scope accepted")
	}
	if err := m.UpdateScope(context.Background(), []string{"https://fixture.local"}, true); err != nil {
		t.Fatal(err)
	}
	if err := m.AddInstructions(context.Background(), "focus on IDOR"); err != nil {
		t.Fatal(err)
	}
	approval := domain.Approval{ID: domain.MustNewID(), ActionID: domain.MustNewID(), State: domain.ApprovalAllowed}
	if err := m.DecideApproval(context.Background(), approval); err != nil {
		t.Fatal(err)
	}
	if err := m.Cancel(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.Create(context.Background(), "Assess API", []string{"https://fixture.local"}, []string{"auth"}); err != nil {
		t.Fatal(err)
	}
	if err := m.SyncEvents(context.Background()); err != nil || m.Cursor != 1 || !strings.Contains(m.View(), "session.created") {
		t.Fatalf("events=%v cursor=%d err=%v", m.Events, m.Cursor, err)
	}
}
