package generator

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConflictResolution represents what to do with an existing file
type ConflictResolution int

const (
	Skip ConflictResolution = iota
	Overwrite
	ShowDiff
	Cancel
)

// Resolver handles file conflict resolution with beautiful UX
type Resolver struct {
	strategy ConflictStrategy
	diffGen  *DiffGenerator // Reused across conflicts
}

// ConflictStrategy determines how to resolve conflicts
type ConflictStrategy interface {
	Resolve(path string, existing, newer []byte) (ConflictResolution, error)
}

// Lipgloss styles for terminal output
var (
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Bold(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	borderStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("white")).Bold(true)
)

// NewResolver creates a conflict resolver with the specified flags.
// Returns error if --force is combined with --skip or --diff.
func NewResolver(force, skip, diff bool) (*Resolver, error) {
	if force && (skip || diff) {
		return nil, fmt.Errorf("--force cannot be combined with --skip or --diff")
	}

	return &Resolver{
		strategy: selectStrategy(force, skip, diff),
		diffGen:  NewDiffGenerator(),
	}, nil
}

// ResolveConflict determines what to do with a file that already exists.
// Returns the user's decision (or automatic decision based on flags).
func (r *Resolver) ResolveConflict(path string, existing, newer []byte) (ConflictResolution, error) {
	return r.strategy.Resolve(path, existing, newer)
}

// selectStrategy chooses the appropriate strategy based on flags
func selectStrategy(force, skip, diff bool) ConflictStrategy {
	switch {
	case force:
		return &ForceStrategy{}
	case skip:
		return &SkipStrategy{}
	case diff:
		return &DiffStrategy{
			diffGen: NewDiffGenerator(),
		}
	default:
		return &InteractiveStrategy{}
	}
}

// ForceStrategy always returns Overwrite (no prompts)
type ForceStrategy struct{}

// Resolve always returns Overwrite for force mode
func (s *ForceStrategy) Resolve(path string, existing, newer []byte) (ConflictResolution, error) {
	return Overwrite, nil
}

// SkipStrategy always returns Skip (no prompts)
type SkipStrategy struct{}

// Resolve always returns Skip for skip mode
func (s *SkipStrategy) Resolve(path string, existing, newer []byte) (ConflictResolution, error) {
	return Skip, nil
}

// DiffStrategy shows diff then delegates to interactive
type DiffStrategy struct {
	diffGen *DiffGenerator
}

// Resolve shows the diff and then prompts for decision
func (s *DiffStrategy) Resolve(path string, existing, newer []byte) (ConflictResolution, error) {
	// Generate diff
	diff := s.diffGen.GenerateDiffDefault(path, path, existing, newer)

	// Count lines in diff
	lineCount := strings.Count(diff, "\n")

	if lineCount > 20 {
		// Show in full-screen viewport
		model := newDiffViewerModel(path, diff)
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return Cancel, fmt.Errorf("failed to show diff: %w", err)
		}

		// Check if user cancelled
		if finalModel.(diffViewerModel).cancelled {
			return Cancel, nil
		}
	} else {
		// Print diff inline for small diffs
		fmt.Println(diff)
	}

	// Now show interactive menu for decision
	interactive := &InteractiveStrategy{}
	return interactive.Resolve(path, existing, newer)
}

// InteractiveStrategy shows menu with keyboard navigation
type InteractiveStrategy struct{}

// Resolve shows interactive menu and returns user's choice.
// Note: If the user selects "Show diff and decide", they will see the diff
// and then be presented with the menu again. This allows them to review the
// diff multiple times if needed before making a decision.
func (s *InteractiveStrategy) Resolve(path string, existing, newer []byte) (ConflictResolution, error) {
	// Get file info
	fileInfo, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return Cancel, fmt.Errorf("failed to stat file: %w", err)
	}

	model := newConflictMenuModel(path, fileInfo)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return Cancel, fmt.Errorf("failed to show menu: %w", err)
	}

	result := finalModel.(conflictMenuModel)
	if result.selected == nil {
		return Cancel, nil
	}

	return *result.selected, nil
}

// conflictMenuModel is the BubbleTea model for the conflict menu
type conflictMenuModel struct {
	path     string
	fileInfo os.FileInfo
	choices  []string
	cursor   int
	selected *ConflictResolution
}

// newConflictMenuModel creates a new conflict menu model
func newConflictMenuModel(path string, fileInfo os.FileInfo) conflictMenuModel {
	return conflictMenuModel{
		path:     path,
		fileInfo: fileInfo,
		choices: []string{
			"Show diff and decide",
			"Skip (keep existing file)",
			"Overwrite (replace with generated code)",
			"Cancel operation",
		},
		cursor: 0,
	}
}

// Init initializes the menu model
func (m conflictMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input
func (m conflictMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter":
			// Map selection to resolution
			resolution := mapChoiceToResolution(m.cursor)
			m.selected = &resolution
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the menu
func (m conflictMenuModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(warningStyle.Render("⚠️  File conflict detected: ") + titleStyle.Render(m.path) + "\n")

	// File info
	if m.fileInfo != nil {
		b.WriteString(mutedStyle.Render("    Last modified: ") + formatRelativeTime(m.fileInfo.ModTime()) + "\n")
		b.WriteString(mutedStyle.Render("    Size: ") + formatFileSize(m.fileInfo.Size()) + "\n")
	}

	b.WriteString("\n")

	// Instructions
	b.WriteString(mutedStyle.Render("    [↑/↓] Navigate    [Enter] Select    [q] Cancel") + "\n\n")

	// Choices
	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
			b.WriteString("    " + selectedStyle.Render(cursor+choice) + "\n")
		} else {
			b.WriteString("    " + cursor + choice + "\n")
		}
	}

	return b.String()
}

// mapChoiceToResolution maps cursor position to resolution
func mapChoiceToResolution(cursor int) ConflictResolution {
	switch cursor {
	case 0:
		return ShowDiff
	case 1:
		return Skip
	case 2:
		return Overwrite
	case 3:
		return Cancel
	default:
		return Cancel
	}
}

// diffViewerModel is the BubbleTea model for showing diffs
type diffViewerModel struct {
	path      string
	diff      string
	viewport  viewport.Model
	ready     bool
	cancelled bool
}

// newDiffViewerModel creates a new diff viewer model
func newDiffViewerModel(path, diff string) diffViewerModel {
	return diffViewerModel{
		path: path,
		diff: diff,
	}
}

// Init initializes the diff viewer
func (m diffViewerModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input and window sizing
func (m diffViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "up", "k":
			m.viewport.ScrollUp(1)

		case "down", "j":
			m.viewport.ScrollDown(1)

		case "pgup", "b":
			m.viewport.PageUp()

		case "pgdown", "f", "space":
			m.viewport.PageDown()
		}

	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 2
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-verticalMargin)
			m.viewport.SetContent(m.diff)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - verticalMargin
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the diff viewer
func (m diffViewerModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	// Header
	title := fmt.Sprintf("─ Diff: %s ", m.path)
	padding := strings.Repeat("─", maxInt(0, m.viewport.Width-len(title)+4))
	b.WriteString(borderStyle.Render(fmt.Sprintf("┌%s%s┐\n", title, padding)))

	// Viewport content
	lines := strings.Split(m.viewport.View(), "\n")
	for _, line := range lines {
		b.WriteString(borderStyle.Render("│") + " " + line)
		// Pad to viewport width
		padding := strings.Repeat(" ", maxInt(0, m.viewport.Width-len(line)-1))
		b.WriteString(padding + borderStyle.Render("│") + "\n")
	}

	// Footer
	footer := " [↑/↓] Scroll    [q] Return to menu "
	padding = strings.Repeat("─", maxInt(0, m.viewport.Width-len(footer)+4))
	b.WriteString(borderStyle.Render(fmt.Sprintf("└%s%s┘\n", padding, footer)))

	return b.String()
}

// Helper functions

// formatRelativeTime formats a time as relative (e.g., "2 hours ago")
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(duration.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
