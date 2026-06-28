package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ncxton/potaco/internal/adapter"
)

var (
	modelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	modelFocusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	modelMutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	modelEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

// searchModel is a custom Bubble Tea model for searching and selecting
// from a list of models by typing. The filter updates in real-time as
// the user types characters.
type searchModel struct {
	providerName string
	models       []adapter.Model
	filtered     []adapter.Model
	cursor       int
	query        textinput.Model
	selected     string
	quitted      bool
}

// newSearchModel creates a new searchModel initialized with the given
// models. All models are shown initially (no filter applied).
func newSearchModel(providerName string, models []adapter.Model) *searchModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type to search..."
	ti.Focus()

	return &searchModel{
		providerName: providerName,
		models:       models,
		filtered:     models,
		query:        ti,
	}
}

// Init implements tea.Model.
func (m *searchModel) Init() tea.Cmd {
	return textinput.Blink
}

// applyFilter updates m.filtered based on the current query value.
// Filtering is case-insensitive and matches against both ID and DisplayName.
func (m *searchModel) applyFilter() {
	q := strings.ToLower(m.query.Value())
	if q == "" {
		m.filtered = m.models
		m.clampCursor()
		return
	}
	m.filtered = nil
	for _, model := range m.models {
		if strings.Contains(strings.ToLower(model.ID), q) ||
			strings.Contains(strings.ToLower(model.DisplayName), q) {
			m.filtered = append(m.filtered, model)
		}
	}
	m.clampCursor()
}

// clampCursor ensures the cursor is within bounds of the filtered list.
func (m *searchModel) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > len(m.filtered)-1 {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// Update implements tea.Model.
func (m *searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.quitted = true
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				m.selected = m.filtered[m.cursor].ID
			}
			return m, tea.Quit
		case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
			if len(m.filtered) > 0 {
				m.cursor = (m.cursor + 1) % len(m.filtered)
			}
		case tea.KeyUp, tea.KeyCtrlK, tea.KeyCtrlP:
			if len(m.filtered) > 0 {
				m.cursor = (m.cursor - 1 + len(m.filtered)) % len(m.filtered)
			}
		default:
			// Any other key: let the text input handle it, then re-filter.
			var cmd tea.Cmd
			m.query, cmd = m.query.Update(msg)
			m.applyFilter()
			return m, cmd
		}
		return m, nil
	default:
		// Forward non-key messages to the text input (e.g. blink).
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(msg)
		return m, cmd
	}
}

// View implements tea.Model.
func (m *searchModel) View() string {
	var b strings.Builder

	b.WriteString(modelTitleStyle.Render("Select a model for " + m.providerName))
	b.WriteString("\n\n")

	b.WriteString(m.query.View())
	b.WriteString("\n\n")

	for i, model := range m.filtered {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		label := model.ID
		if model.DisplayName != "" && model.DisplayName != model.ID {
			label += "  " + model.DisplayName
		}
		if i == m.cursor {
			label = modelFocusStyle.Render(label)
		}
		b.WriteString(cursor)
		b.WriteString(label)
		b.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(modelEmptyStyle.Render("  No matching models."))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(modelMutedStyle.Render("  \u2191\u2193 navigate  enter select  esc cancel"))
	b.WriteString("\n")

	return b.String()
}
