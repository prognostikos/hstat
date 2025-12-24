package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/betternow/hstat/store"
	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	if m.width < 40 || m.height < 15 {
		return "Terminal too small (need 40x15 minimum)"
	}

	if m.showHelp {
		return m.renderHelp()
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Stats
	b.WriteString(m.renderStats())
	b.WriteString("\n\n")

	// Status codes
	b.WriteString(m.renderStatusCodes())
	b.WriteString("\n")

	// Hosts and IPs - side by side if wide enough
	if m.width >= 100 {
		b.WriteString(m.renderHostsAndIPsSideBySide())
	} else {
		b.WriteString(m.renderHosts())
		b.WriteString("\n")
		b.WriteString(m.renderIPs())
	}

	// Footer with help hint
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("? help  w whois  i ipinfo  q quit"))

	content := b.String()

	// Render modal overlay if visible
	if m.modal.Visible {
		content = m.renderWithModal(content)
	}

	return content
}

func (m Model) renderHeader() string {
	elapsed := time.Since(m.startTime).Round(time.Second)

	status := "hstat"
	if m.streamEnded {
		status = "hstat (stream ended)"
	}

	header := fmt.Sprintf("%s | %s | %s reqs | %.1f/s avg",
		status,
		elapsed,
		formatNumber(m.stats.TotalCount),
		float64(m.stats.TotalCount)/float64(max64(1, int64(elapsed.Seconds()))),
	)

	result := headerStyle.Render(header)

	// Filter indicator
	if m.filter.Host != "" {
		result += "  " + filterStyle.Render(fmt.Sprintf("[host=%s] Esc to clear", m.filter.Host))
	} else if m.filter.IP != "" {
		result += "  " + filterStyle.Render(fmt.Sprintf("[ip=%s] Esc to clear", m.filter.IP))
	}

	return result
}

func (m Model) renderStats() string {
	respLine := fmt.Sprintf("%s  avg %s  |  p50 %s  |  p95 %s  |  p99 %s  |  max %s",
		statsLabelStyle.Render("Response (ms)"),
		statsValueStyle.Render(fmt.Sprintf("%5d", m.stats.AvgService)),
		statsValueStyle.Render(fmt.Sprintf("%5d", m.stats.P50Service)),
		statsValueStyle.Render(fmt.Sprintf("%5d", m.stats.P95Service)),
		statsValueStyle.Render(fmt.Sprintf("%5d", m.stats.P99Service)),
		statsValueStyle.Render(fmt.Sprintf("%5d", m.stats.MaxService)),
	)

	connLine := fmt.Sprintf("%s   avg %s  |  max %s",
		statsLabelStyle.Render("Connect (ms)"),
		statsValueStyle.Render(fmt.Sprintf("%5d", m.stats.AvgConnect)),
		statsValueStyle.Render(fmt.Sprintf("%5d", m.stats.MaxConnect)),
	)

	return respLine + "\n" + connLine
}

func (m Model) renderStatusCodes() string {
	var b strings.Builder

	title := sectionTitleStyle.Render("Status Codes")
	b.WriteString(title)
	b.WriteString("\n")

	if len(m.statusCounts) == 0 {
		b.WriteString(tableRowDimStyle.Render("  No data"))
		return b.String()
	}

	// Calculate total for percentages
	var total int64
	for _, sc := range m.statusCounts {
		total += sc.Count
	}

	for _, sc := range m.statusCounts {
		pct := float64(sc.Count) * 100 / float64(max64(1, total))
		statusStr := StatusStyle(sc.Status).Render(fmt.Sprintf("%d", sc.Status))
		line := fmt.Sprintf("  %s  %8s  %5.1f%%", statusStr, formatNumber(sc.Count), pct)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderHosts() string {
	active := m.section == SectionHosts
	// Dim hosts when filtering BY host (host is the filter source)
	return m.renderList("Hosts", m.topHosts, m.otherHosts, m.hostCursor, active, m.filter.Host != "")
}

func (m Model) renderIPs() string {
	active := m.section == SectionIPs
	// Dim IPs when filtering BY IP (IP is the filter source)
	return m.renderList("IPs", m.topIPs, m.otherIPs, m.ipCursor, active, m.filter.IP != "")
}

func (m Model) renderList(title string, items []store.CountItem, other int64, cursor int, active bool, dimmed bool) string {
	var b strings.Builder

	titleStyle := sectionTitleStyle
	if active {
		titleStyle = sectionTitleActiveStyle
	}

	b.WriteString(titleStyle.Render(title))
	if active {
		b.WriteString(" " + helpStyle.Render("[j/k Enter]"))
	}
	b.WriteString("\n")

	if len(items) == 0 {
		style := tableRowDimStyle
		b.WriteString(style.Render("  No data"))
		return b.String()
	}

	// Calculate total for percentages
	var total int64
	for _, item := range items {
		total += item.Count
	}
	total += other

	maxLabelLen := 30
	for i, item := range items {
		label := item.Label
		if len(label) > maxLabelLen {
			label = label[:maxLabelLen-3] + "..."
		}

		pct := float64(item.Count) * 100 / float64(max64(1, total))
		line := fmt.Sprintf("%-*s  %8s  %5.1f%%", maxLabelLen, label, formatNumber(item.Count), pct)

		var style lipgloss.Style
		if dimmed {
			style = tableRowDimStyle
		} else if active && i == cursor {
			line = "> " + line
			style = tableRowSelectedStyle
		} else {
			line = "  " + line
			style = tableRowStyle
		}

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Other
	if other > 0 {
		pct := float64(other) * 100 / float64(max64(1, total))
		line := fmt.Sprintf("  %-*s  %8s  %5.1f%%", maxLabelLen, "(other)", formatNumber(other), pct)
		b.WriteString(tableRowDimStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderHostsAndIPsSideBySide() string {
	hosts := m.renderHosts()
	ips := m.renderIPs()

	// Split into lines and join side by side
	hostLines := strings.Split(hosts, "\n")
	ipLines := strings.Split(ips, "\n")

	colWidth := (m.width - 4) / 2

	var b strings.Builder
	maxLines := max(len(hostLines), len(ipLines))

	for i := 0; i < maxLines; i++ {
		var hostLine, ipLine string
		if i < len(hostLines) {
			hostLine = hostLines[i]
		}
		if i < len(ipLines) {
			ipLine = ipLines[i]
		}

		// Pad host line to column width
		hostLine = padRight(hostLine, colWidth)

		b.WriteString(hostLine)
		b.WriteString("  ")
		b.WriteString(ipLine)
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderHelp() string {
	help := `
  hstat - Heroku Router Log Monitor

  Navigation:
    Tab / l        Next section
    Shift+Tab / h  Previous section
    j / Down       Move cursor down
    k / Up         Move cursor up
    g              Jump to top
    G              Jump to bottom

  Actions:
    Enter          Filter by selected host/IP
    w              Whois lookup (when IP selected)
    i              IP info lookup via ipinfo.io (when IP selected)
    Esc            Clear filter (or quit if no filter)
    q / Ctrl+C     Quit

  Press any key to dismiss this help
`
	return help
}

func (m Model) renderWithModal(background string) string {
	// Calculate modal dimensions
	modalWidth := min(m.width-4, 80)
	modalHeight := min(m.height-4, 30)

	// Build modal content
	var content strings.Builder

	// Title
	content.WriteString(modalTitleStyle.Render(m.modal.Title))
	content.WriteString("\n")

	// Content - truncate if too long
	modalContent := m.modal.Content
	lines := strings.Split(modalContent, "\n")

	// Reserve space for title and hint
	maxContentLines := modalHeight - 4
	if len(lines) > maxContentLines {
		lines = lines[:maxContentLines]
		lines = append(lines, "... (truncated)")
	}

	// Truncate long lines
	for i, line := range lines {
		if lipgloss.Width(line) > modalWidth-6 {
			lines[i] = line[:modalWidth-9] + "..."
		}
	}

	content.WriteString(modalContentStyle.Render(strings.Join(lines, "\n")))
	content.WriteString("\n")
	content.WriteString(modalHintStyle.Render("Press Esc or Enter to close"))

	// Style the modal box
	modal := modalStyle.
		Width(modalWidth).
		Render(content.String())

	// Center the modal based on terminal size (not background content)
	modalLines := strings.Split(modal, "\n")

	// Calculate vertical position (center based on terminal height)
	startY := (m.height - len(modalLines)) / 2
	if startY < 0 {
		startY = 0
	}

	// Calculate horizontal position (center based on terminal width)
	modalLineWidth := 0
	if len(modalLines) > 0 {
		modalLineWidth = lipgloss.Width(modalLines[0])
	}
	startX := (m.width - modalLineWidth) / 2
	if startX < 0 {
		startX = 0
	}

	// Build result with fixed terminal height
	bgLines := strings.Split(background, "\n")
	result := make([]string, m.height)

	for i := 0; i < m.height; i++ {
		// Get background line (or empty if beyond background content)
		var bgLine string
		if i < len(bgLines) {
			bgLine = bgLines[i]
		}

		if i >= startY && i < startY+len(modalLines) {
			modalLineIdx := i - startY
			modalLine := modalLines[modalLineIdx]
			currentModalWidth := lipgloss.Width(modalLine)

			// Dim the background line
			dimmedBg := tableRowDimStyle.Render(stripAnsi(bgLine))

			// Build the composite line with fixed positioning
			prefix := strings.Repeat(" ", startX)
			if startX > 0 && lipgloss.Width(dimmedBg) >= startX {
				prefix = substring(dimmedBg, 0, startX)
			}

			suffix := ""
			afterModal := startX + currentModalWidth
			if lipgloss.Width(dimmedBg) > afterModal {
				suffix = substring(dimmedBg, afterModal, lipgloss.Width(dimmedBg))
			}

			result[i] = prefix + modalLine + suffix
		} else {
			result[i] = tableRowDimStyle.Render(stripAnsi(bgLine))
		}
	}

	return strings.Join(result, "\n")
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	// Simple approach - remove common ANSI sequences
	result := s
	for strings.Contains(result, "\x1b[") {
		start := strings.Index(result, "\x1b[")
		end := start + 2
		for end < len(result) && result[end] != 'm' {
			end++
		}
		if end < len(result) {
			result = result[:start] + result[end+1:]
		} else {
			break
		}
	}
	return result
}

// substring extracts a visible substring handling ANSI codes
func substring(s string, start, end int) string {
	// For simplicity, strip ANSI and pad
	plain := stripAnsi(s)
	if start >= len(plain) {
		return ""
	}
	if end > len(plain) {
		end = len(plain)
	}
	return plain[start:end]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper functions

func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func padRight(s string, width int) string {
	// Use lipgloss.Width() to measure visible width (handles ANSI codes)
	visibleWidth := lipgloss.Width(s)
	if visibleWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visibleWidth)
}
