package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type Client interface {
	Call(context.Context, string, any, any) error
}
type Screen int

const (
	Overview Screen = iota
	SessionDetail
	ApprovalDetail
	InstructionEditor
	ScopeEditor
	CreateEditor
	SearchEditor
)

type Model struct {
	Client        Client
	Width, Height int
	Screen        Screen
	Sessions      []domain.Session
	Selected      int
	Query         string
	Current       domain.Session
	Status        string
	Connected     bool
	Events        []domain.Event
	Findings      []domain.Finding
	CoverageGaps  []domain.CoverageGap
	Cursor        uint64
}

func (m *Model) Refresh(ctx context.Context) error {
	var sessions []domain.Session
	if err := m.Client.Call(ctx, "session.list", nil, &sessions); err != nil {
		m.Connected = false
		m.Status = err.Error()
		return err
	}
	m.Sessions, m.Connected = sessions, true
	return nil
}

func (m *Model) AddInstructions(ctx context.Context, value string) error {
	var updated domain.Session
	if err := m.Client.Call(ctx, "session.instructions.update", map[string]any{"id": m.Current.ID, "instructions": value}, &updated); err != nil {
		return err
	}
	m.Current, m.Screen = updated, SessionDetail
	return nil
}

func (m *Model) UpdateScope(ctx context.Context, targets []string, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("scope update requires explicit confirmation")
	}
	var updated domain.Session
	if err := m.Client.Call(ctx, "session.scope.update", map[string]any{"id": m.Current.ID, "targets": targets, "confirmed": true}, &updated); err != nil {
		return err
	}
	m.Current, m.Screen = updated, SessionDetail
	return nil
}

func (m *Model) DecideApproval(ctx context.Context, approval domain.Approval) error {
	var updated domain.Session
	if err := m.Client.Call(ctx, "approval.decide", map[string]any{"session_id": m.Current.ID, "approval": approval}, &updated); err != nil {
		return err
	}
	m.Current, m.Screen = updated, SessionDetail
	return nil
}

func (m *Model) Cancel(ctx context.Context) error {
	var updated domain.Session
	if err := m.Client.Call(ctx, "session.cancel", map[string]any{"id": m.Current.ID, "reason": "cancelled from TUI"}, &updated); err != nil {
		return err
	}
	m.Current = updated
	return nil
}

func (m *Model) Create(ctx context.Context, objective string, targets, goals []string) error {
	var created domain.Session
	if err := m.Client.Call(ctx, "session.create", map[string]any{"objective": objective, "targets": targets, "goals": goals}, &created); err != nil {
		return err
	}
	m.Sessions = append([]domain.Session{created}, m.Sessions...)
	m.Current, m.Screen = created, SessionDetail
	return nil
}

func (m *Model) SyncEvents(ctx context.Context) error {
	if m.Current.ID.Validate() != nil {
		// The detail screen may render before a session is selected.
		//nolint:nilerr
		return nil
	}
	var events []domain.Event
	if err := m.Client.Call(ctx, "session.events", map[string]any{"id": m.Current.ID, "after": m.Cursor}, &events); err != nil {
		m.Connected = false
		return err
	}
	for _, event := range events {
		m.Events = append(m.Events, event)
		m.Cursor = event.Sequence
	}
	var findings []json.RawMessage
	if err := m.Client.Call(ctx, "session.records", map[string]any{"id": m.Current.ID, "kind": "finding"}, &findings); err != nil {
		m.Connected = false
		return err
	}
	m.Findings = decodeRecords[domain.Finding](findings)
	var gaps []json.RawMessage
	if err := m.Client.Call(ctx, "session.records", map[string]any{"id": m.Current.ID, "kind": "coverage-gap"}, &gaps); err != nil {
		m.Connected = false
		return err
	}
	m.CoverageGaps = decodeRecords[domain.CoverageGap](gaps)
	m.Connected = true
	return nil
}

func New(client Client) Model {
	return Model{Client: client, Width: 80, Height: 24, Connected: client != nil}
}
func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch value := message.(type) {
	case tea.WindowSizeMsg:
		m.Width, m.Height = value.Width, value.Height
	case tea.KeyMsg:
		switch value.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "down", "j":
			if m.Selected < len(m.visible())-1 {
				m.Selected++
			}
		case "up", "k":
			if m.Selected > 0 {
				m.Selected--
			}
		case "enter":
			rows := m.visible()
			if len(rows) > 0 {
				m.Current = rows[m.Selected]
				m.Screen = SessionDetail
			}
		case "esc":
			m.Screen = Overview
		case "i":
			if m.Screen == SessionDetail {
				m.Screen = InstructionEditor
			}
		case "a":
			if m.Screen == SessionDetail {
				m.Screen = ApprovalDetail
			}
		case "s":
			if m.Screen == SessionDetail {
				m.Screen = ScopeEditor
			}
		case "n":
			if m.Screen == Overview {
				m.Screen = CreateEditor
			}
		case "/":
			if m.Screen == Overview {
				m.Screen = SearchEditor
			}
		}
	}
	return m, nil
}
func (m Model) View() string {
	var body string
	if !m.Connected {
		body = "RECONNECTING\n\nWaiting for the local CyberPilot runtime..."
	} else {
		switch m.Screen {
		case Overview:
			body = m.overview()
		case SessionDetail:
			body = m.session()
		case ApprovalDetail:
			body = "APPROVAL\n\nReview purpose, target, risk, side effects, and limits.\n\n[a] allow  [r] restrict  [d] deny  [esc] back"
		case InstructionEditor:
			body = "ADD INSTRUCTIONS\n\nInstructions guide replanning but never expand scope.\n\n[enter] save  [esc] cancel"
		case ScopeEditor:
			body = "UPDATE SCOPE\n\nScope changes require explicit confirmation.\n\n[enter] confirm  [esc] cancel"
		case CreateEditor:
			body = "NEW SESSION\n\nEnter one objective with explicit targets and goals.\n\n[enter] create  [esc] cancel"
		case SearchEditor:
			body = "SEARCH SESSIONS\n\nType to filter by session name or objective.\n\n[enter] apply  [esc] cancel"
		}
	}
	return constrain(body, m.Width, m.Height)
}
func (m Model) visible() []domain.Session {
	query := strings.ToLower(strings.TrimSpace(m.Query))
	if query == "" {
		return m.Sessions
	}
	var result []domain.Session
	for _, session := range m.Sessions {
		if strings.Contains(strings.ToLower(session.Name+" "+session.Objective), query) {
			result = append(result, session)
		}
	}
	return result
}
func (m Model) overview() string {
	needs, other := splitSessions(m.visible())
	var b strings.Builder
	b.WriteString("CYBERPILOT  SESSIONS\n")
	fmt.Fprintf(&b, "%d active  %d needs input  %d total\n\n", countActive(m.Sessions), len(needs), len(m.Sessions))
	renderList(&b, "NEEDS INPUT", needs, m.Width)
	b.WriteString("\n")
	renderList(&b, "OTHER SESSIONS", other, m.Width)
	b.WriteString("\n[n] new  [/] search  [enter] open  [q] quit")
	return b.String()
}
func (m Model) session() string {
	s := m.Current
	timeline := "No records yet."
	if len(m.Events) > 0 {
		var lines []string
		for _, event := range m.Events {
			lines = append(lines, fmt.Sprintf("#%d %s", event.Sequence, event.Type))
		}
		timeline = strings.Join(lines, "\n")
	}
	return fmt.Sprintf("SESSION  %s\n\nObjective\n%s\n\nTargets\n%s\n\nGoals\n%s\n\nState: %s\nVerified findings: %d\nCoverage gaps: %d\n\nTIMELINE / EVIDENCE / FINDINGS / COVERAGE GAPS\n%s\n\n[i] instruct  [a] approval  [s] scope  [c] cancel  [esc] back", s.Name, wrap(s.Objective, m.Width-2), wrap(strings.Join(s.Targets, "\n"), m.Width-2), wrap(strings.Join(s.Goals, "\n"), m.Width-2), s.State, len(m.Findings), len(m.CoverageGaps), timeline)
}

func decodeRecords[T any](raw []json.RawMessage) []T {
	result := make([]T, 0, len(raw))
	for _, data := range raw {
		var value T
		if json.Unmarshal(data, &value) == nil {
			result = append(result, value)
		}
	}
	return result
}
func splitSessions(sessions []domain.Session) ([]domain.Session, []domain.Session) {
	var needs, other []domain.Session
	for _, s := range sessions {
		if s.State == domain.SessionNeedsInput {
			needs = append(needs, s)
		} else {
			other = append(other, s)
		}
	}
	return needs, other
}
func countActive(sessions []domain.Session) int {
	count := 0
	for _, s := range sessions {
		if s.State == domain.SessionCreated || s.State == domain.SessionRunning {
			count++
		}
	}
	return count
}
func renderList(b *strings.Builder, title string, sessions []domain.Session, width int) {
	b.WriteString(title + "\n")
	if len(sessions) == 0 {
		b.WriteString("  None\n")
		return
	}
	for _, s := range sessions {
		line := fmt.Sprintf("  %-12s  %s", strings.ToUpper(string(s.State)), ellipsis(s.Name, max(8, width-18)))
		b.WriteString(line + "\n")
	}
}
func constrain(value string, width, height int) string {
	if width < 1 {
		return ""
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = ellipsis(line, width)
	}
	if height > 0 && len(lines) > height {
		lines = lines[:height]
		lines[height-1] = ellipsis("...", width)
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}
func ellipsis(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}
func wrap(value string, width int) string {
	if width <= 0 {
		return ""
	}
	var lines []string
	for _, source := range strings.Split(value, "\n") {
		for utf8.RuneCountInString(source) > width {
			r := []rune(source)
			lines = append(lines, string(r[:width]))
			source = string(r[width:])
		}
		lines = append(lines, source)
	}
	return strings.Join(lines, "\n")
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
