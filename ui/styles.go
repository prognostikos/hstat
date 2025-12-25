package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("39")  // Blue
	secondaryColor = lipgloss.Color("245") // Gray
	accentColor    = lipgloss.Color("170") // Purple
	successColor   = lipgloss.Color("42")  // Green
	warningColor   = lipgloss.Color("214") // Orange
	errorColor     = lipgloss.Color("196") // Red
	dimColor       = lipgloss.Color("240") // Dim gray

	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Filter indicator
	filterStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// Stats
	statsLabelStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	statsValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)

	// Section titles
	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor)

	sectionTitleActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor)

	// Table
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	tableRowStyle = lipgloss.NewStyle()

	tableRowSelectedStyle = lipgloss.NewStyle().
				Bold(true) // Bold only, no underline per spec

	tableRowDimStyle = lipgloss.NewStyle().
				Foreground(dimColor)

	// Status code colors
	status1xxStyle = lipgloss.NewStyle().Foreground(secondaryColor) // Informational
	status2xxStyle = lipgloss.NewStyle().Foreground(successColor)
	status3xxStyle = lipgloss.NewStyle().Foreground(primaryColor)
	status4xxStyle = lipgloss.NewStyle().Foreground(warningColor)
	status5xxStyle = lipgloss.NewStyle().Foreground(errorColor)

	// Cursor
	cursorStyle = lipgloss.NewStyle().Foreground(accentColor).Bold(true)

	// Help
	helpStyle = lipgloss.NewStyle().Foreground(secondaryColor)

	// Warning styles
	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	streamEndedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	// Borders for sections - using sharp corners (NormalBorder)
	sectionStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(secondaryColor).
			Padding(0, 1)

	sectionActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(accentColor).
				Padding(0, 1)

	// Section border styles for the new layout
	sectionBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(secondaryColor)

	sectionActiveBorderStyle = lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(accentColor) // Magenta/purple for active

	// Modal styles - sharp corners
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(accentColor).
			Padding(1, 2)

	modalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(accentColor)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	modalContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	modalHintStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			MarginTop(1)

	// Trend indicators
	trendUpStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	trendDownStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	// Error rate inline
	errorRateStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	errorRateHighStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	// Count badge style
	countBadgeStyle = lipgloss.NewStyle().
			Foreground(dimColor)
)

// StatusStyle returns the appropriate style for a status code
func StatusStyle(status int) lipgloss.Style {
	switch {
	case status >= 500:
		return status5xxStyle
	case status >= 400:
		return status4xxStyle
	case status >= 300:
		return status3xxStyle
	case status >= 200:
		return status2xxStyle
	default:
		return status1xxStyle
	}
}

// StatusCategoryStyle returns the style for a status code category (1xx, 2xx, etc.)
func StatusCategoryStyle(category int) lipgloss.Style {
	switch category {
	case 1:
		return status1xxStyle
	case 2:
		return status2xxStyle
	case 3:
		return status3xxStyle
	case 4:
		return status4xxStyle
	case 5:
		return status5xxStyle
	default:
		return lipgloss.NewStyle()
	}
}

// RenderSection renders content within a bordered section
func RenderSection(title, content string, width int, active bool) string {
	style := sectionBorderStyle
	if active {
		style = sectionActiveBorderStyle
	}

	// Build the section with title in the border
	titleStr := "─ " + title + " "

	// Create a custom border with title
	border := lipgloss.NormalBorder()

	styledContent := style.
		Width(width - 2). // Account for border
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Render(content)

	// Replace top border with title
	lines := splitLines(styledContent)
	if len(lines) > 0 && len(lines[0]) > 3 {
		remaining := width - len(titleStr) - 2
		if remaining < 0 {
			remaining = 0
		}
		newTop := string(border.TopLeft) + titleStr
		for i := 0; i < remaining; i++ {
			newTop += string(border.Top)
		}
		newTop += string(border.TopRight)
		lines[0] = newTop
	}

	return joinLines(lines)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
