package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ncxton/potaco/internal/adapter"
)

func TestNewSearchModel(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2", SupportsEdit: true},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)

	if len(m.models) != 3 {
		t.Errorf("expected 3 models, got %d", len(m.models))
	}
	if len(m.filtered) != 3 {
		t.Errorf("expected 3 filtered models initially, got %d", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
}

func TestSearchFilterByID(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.query.SetValue("gpt")

	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered model, got %d", len(m.filtered))
	}
	if m.filtered[0].ID != "gpt-image-2" {
		t.Errorf("expected gpt-image-2, got %s", m.filtered[0].ID)
	}
}

func TestSearchFilterByDisplayName(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.query.SetValue("dall")

	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered model, got %d", len(m.filtered))
	}
	if m.filtered[0].ID != "dall-e-3" {
		t.Errorf("expected dall-e-3, got %s", m.filtered[0].ID)
	}
}

func TestSearchFilterCaseInsensitive(t *testing.T) {
	models := []adapter.Model{
		{ID: "GPT-Image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
	}
	m := newSearchModel(models)
	m.query.SetValue("gpt")

	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered model (case-insensitive), got %d", len(m.filtered))
	}
}

func TestSearchFilterEmptyShowsAll(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
	}
	m := newSearchModel(models)
	m.query.SetValue("")

	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered when query empty, got %d", len(m.filtered))
	}
}

func TestSearchFilterNoMatch(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
	}
	m := newSearchModel(models)
	m.query.SetValue("xyz123")

	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 filtered models, got %d", len(m.filtered))
	}
}

func TestSearchCursorClampsToFiltered(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.cursor = 2 // at last item

	// Filter to only 1 item
	m.query.SetValue("flux")
	m.applyFilter()

	// applyFilter calls clampCursor internally, so cursor should be 0
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.cursor)
	}
}

func TestSearchEscQuits(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
	}
	m := newSearchModel(models)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if !m.quitted {
		t.Error("expected quitted=true after Esc")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
}

func TestSearchEnterSelects(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
	}
	m := newSearchModel(models)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
	// Note: cursor stays at 0 when filter is non-empty and unchanged.
	// Update returns the current state before selection assignment completes.
	// The first filtered model is gpt-image-2.
	if m.selected != "gpt-image-2" {
		t.Errorf("expected selected gpt-image-2, got %q", m.selected)
	}
}

func TestSearchArrowDownMovesCursor(t *testing.T) {
	models := []adapter.Model{
		{ID: "gpt-image-2", DisplayName: "GPT Image 2"},
		{ID: "dall-e-3", DisplayName: "DALL-E 3"},
		{ID: "flux-pro", DisplayName: "Flux Pro"},
	}
	m := newSearchModel(models)
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", m.cursor)
	}
	// Wrap at bottom
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 0 {
		t.Errorf("expected cursor to wrap to 0, got %d", m.cursor)
	}
}
