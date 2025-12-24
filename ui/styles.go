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
				Foreground(lipgloss.Color("255")).
				Background(primaryColor)

	tableRowDimStyle = lipgloss.NewStyle().
				Foreground(dimColor)

	// Status code colors
	status2xxStyle = lipgloss.NewStyle().Foreground(successColor)
	status3xxStyle = lipgloss.NewStyle().Foreground(primaryColor)
	status4xxStyle = lipgloss.NewStyle().Foreground(warningColor)
	status5xxStyle = lipgloss.NewStyle().Foreground(errorColor)

	// Cursor
	cursorStyle = lipgloss.NewStyle().Foreground(accentColor).Bold(true)

	// Help
	helpStyle = lipgloss.NewStyle().Foreground(secondaryColor)

	// Borders for sections
	sectionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(0, 1)

	sectionActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Padding(0, 1)

	// Modal styles
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	modalContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	modalHintStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			MarginTop(1)
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
	default:
		return status2xxStyle
	}
}
