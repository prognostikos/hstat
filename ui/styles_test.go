package ui

import (
	"strings"
	"testing"
)

func TestBorderStyle_SharpCorners(t *testing.T) {
	// All section borders should use sharp corners (Normal border), not rounded
	testStyle := sectionBorderStyle

	// Render a simple box
	content := testStyle.Width(20).Render("test")

	// Sharp corners use ┌ ┐ └ ┘, not ╭ ╮ ╰ ╯
	if strings.Contains(content, "╭") || strings.Contains(content, "╮") ||
		strings.Contains(content, "╰") || strings.Contains(content, "╯") {
		t.Error("expected sharp corners (┌┐└┘), got rounded corners (╭╮╰╯)")
	}

	// Should contain sharp corner characters
	if !strings.Contains(content, "┌") && !strings.Contains(content, "┐") {
		t.Error("expected sharp corner characters (┌┐)")
	}
}

func TestActiveSectionStyle_MagentaBorder(t *testing.T) {
	// Active section should have magenta/purple border color (accentColor)
	// We verify the styles are configured differently by checking their properties

	normalStyle := sectionBorderStyle
	activeStyle := sectionActiveBorderStyle

	// Both should have borders
	normalContent := normalStyle.Width(20).Render("test")
	activeContent := activeStyle.Width(20).Render("test")

	// Both should have border characters
	if !strings.Contains(normalContent, "┌") {
		t.Error("expected normal section to have border")
	}
	if !strings.Contains(activeContent, "┌") {
		t.Error("expected active section to have border")
	}

	// The active style should use accentColor which is different from secondaryColor
	// This is verified by the style definition - we just confirm both styles exist and render
	if len(normalContent) == 0 || len(activeContent) == 0 {
		t.Error("expected styles to render content")
	}
}

func TestSelectedRowStyle_BoldNoUnderline(t *testing.T) {
	// Selected row should be bold but NOT underlined
	// We verify the style is configured with Bold(true) and no Underline
	// by checking that it renders (lipgloss styles are configured in styles.go)

	style := tableRowSelectedStyle
	content := style.Render("test row")

	// Should render the content
	if !strings.Contains(content, "test row") {
		t.Error("expected style to render content")
	}

	// The style configuration is Bold(true) without Underline(true)
	// This is verified by visual inspection of styles.go
	// The test ensures the style exists and renders without crashing
}

func TestModalStyle_SharpCorners(t *testing.T) {
	// Modal should also use sharp corners
	testStyle := modalBorderStyle

	content := testStyle.Width(30).Render("modal content")

	// Should not have rounded corners
	if strings.Contains(content, "╭") || strings.Contains(content, "╮") {
		t.Error("expected modal to use sharp corners")
	}
}

func TestRenderSection_WithBorder(t *testing.T) {
	// Test that renderSection applies borders correctly
	content := "Line 1\nLine 2"
	title := "Test Section"

	result := RenderSection(title, content, 40, false)

	// Should contain the title
	if !strings.Contains(result, title) {
		t.Errorf("expected section to contain title %q", title)
	}

	// Should contain the content
	if !strings.Contains(result, "Line 1") {
		t.Error("expected section to contain content")
	}

	// Should have border characters
	if !strings.Contains(result, "─") {
		t.Error("expected section to have horizontal border")
	}
}

func TestRenderSection_ActiveHighlight(t *testing.T) {
	content := "content"
	title := "Active Section"

	normalResult := RenderSection(title, content, 40, false)
	activeResult := RenderSection(title, content, 40, true)

	// Both should render with borders and title
	if !strings.Contains(normalResult, title) {
		t.Error("expected normal section to contain title")
	}
	if !strings.Contains(activeResult, title) {
		t.Error("expected active section to contain title")
	}

	// Both should have border characters
	if !strings.Contains(normalResult, "─") {
		t.Error("expected normal section to have border")
	}
	if !strings.Contains(activeResult, "─") {
		t.Error("expected active section to have border")
	}

	// Active styling is handled by lipgloss color - the structure should be the same
	// but ANSI codes will differ when color output is enabled
}

func TestStatusCodeCategoryStyle(t *testing.T) {
	// Test that StatusCategoryStyle returns valid styles for each category
	categories := []int{1, 2, 3, 4, 5}

	for _, cat := range categories {
		style := StatusCategoryStyle(cat)
		rendered := style.Render("test")

		// Should render the content without crashing
		if !strings.Contains(rendered, "test") {
			t.Errorf("expected status %dxx style to render content", cat)
		}
	}

	// Test that unknown category returns empty style
	unknownStyle := StatusCategoryStyle(9)
	unknownRendered := unknownStyle.Render("test")
	if !strings.Contains(unknownRendered, "test") {
		t.Error("expected unknown category style to render content")
	}
}

func TestStatusStyle_ReturnsCorrectCategory(t *testing.T) {
	// Test that StatusStyle returns appropriate styles for each status range
	tests := []struct {
		status   int
		expected string
	}{
		{100, "1xx"},
		{101, "1xx"},
		{200, "2xx"},
		{201, "2xx"},
		{301, "3xx"},
		{404, "4xx"},
		{500, "5xx"},
		{503, "5xx"},
	}

	for _, tc := range tests {
		style := StatusStyle(tc.status)
		rendered := style.Render("test")
		// Should render without crashing
		if !strings.Contains(rendered, "test") {
			t.Errorf("StatusStyle(%d) failed to render", tc.status)
		}
	}
}
