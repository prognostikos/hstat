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

	if m.width < MinWidth || m.height < MinHeight {
		return fmt.Sprintf("Terminal too small (need %dx%d minimum)", MinWidth, MinHeight)
	}

	// Calculate layout based on terminal dimensions
	layout := CalculateLayoutWithActiveSection(m.width, m.height, m.section)
	if layout == nil {
		return "Terminal too small"
	}

	var sections []string

	// Header section with border
	headerContent := m.renderHeaderContent()
	headerSection := m.renderBorderedSection("hstat", headerContent, m.width, false)
	sections = append(sections, headerSection)

	// Status codes section with border (columnar layout)
	statusData := StatusCodesDataFromStore(m.statusCounts)
	statusContent := RenderStatusCodesColumnar(statusData, m.width-4, layout.StatusCodeColumns)
	statusSection := m.renderBorderedSection("Status Codes", statusContent, m.width, false)
	sections = append(sections, statusSection)

	// Calculate remaining height for data sections
	usedHeight := countLines(headerSection) + countLines(statusSection)
	remainingHeight := m.height - usedHeight

	// Data sections
	dataContent := m.renderDataSections(layout, remainingHeight)
	sections = append(sections, dataContent)

	// Join all sections
	content := strings.Join(sections, "\n")

	// Ensure we don't exceed terminal height
	lines := strings.Split(content, "\n")
	if len(lines) > m.height {
		lines = lines[:m.height]
		content = strings.Join(lines, "\n")
	}

	// Render modal overlay if visible
	if m.modal.Visible {
		content = m.renderWithModal(content)
	}

	return content
}

// renderHeaderContent renders header stats without border
func (m Model) renderHeaderContent() string {
	elapsed := time.Since(m.startTime).Round(time.Second)

	line1 := fmt.Sprintf("%s | %s reqs | %.1f/s",
		elapsed,
		formatNumber(m.stats.TotalCount),
		m.currentRate,
	)

	// Add error rates and trend
	if m.stats.TotalCount > 0 {
		if m.rate4xx > 0 {
			line1 += fmt.Sprintf(" | %s", status4xxStyle.Render(fmt.Sprintf("4xx:%.1f%%", m.rate4xx)))
		}
		if m.rate5xx > 0 {
			line1 += fmt.Sprintf(" %s", status5xxStyle.Render(fmt.Sprintf("5xx:%.1f%%", m.rate5xx)))
		}
		// 1m trend
		switch m.trend {
		case store.TrendUp:
			line1 += " " + trendUpStyle.Render("1m↑")
		case store.TrendDown:
			line1 += " " + trendDownStyle.Render("1m↓")
		}
		// 5m trend
		switch m.trend5m {
		case store.TrendUp:
			line1 += " " + trendUpStyle.Render("5m↑")
		case store.TrendDown:
			line1 += " " + trendDownStyle.Render("5m↓")
		}
	}

	// Stream status
	if m.streamEnded {
		line1 += "  " + streamEndedStyle.Render("⚠ STREAM ENDED")
	} else if !m.lastEntryTime.IsZero() {
		sinceLastEntry := time.Since(m.lastEntryTime)
		if sinceLastEntry > noDataWarningThreshold {
			secs := int(sinceLastEntry.Seconds())
			line1 += "  " + warningStyle.Render(fmt.Sprintf("⚠ no data for %ds", secs))
		}
	}

	// Stats lines
	line2 := fmt.Sprintf("Response: avg %dms | p50 %dms | p95 %dms | p99 %dms | max %dms",
		m.stats.AvgService, m.stats.P50Service, m.stats.P95Service, m.stats.P99Service, m.stats.MaxService)
	line3 := fmt.Sprintf("Connect:  avg %dms | max %dms",
		m.stats.AvgConnect, m.stats.MaxConnect)

	return line1 + "\n" + line2 + "\n" + line3
}

// renderBorderedSection renders content within a bordered box
func (m Model) renderBorderedSection(title, content string, width int, active bool) string {
	borderStyle := sectionBorderStyle
	if active {
		borderStyle = sectionActiveBorderStyle
	}

	// Calculate inner width
	innerWidth := width - 4 // borders + padding

	// Render the content with border
	styled := borderStyle.
		Width(innerWidth).
		Render(content)

	// Replace top border with title
	lines := strings.Split(styled, "\n")
	if len(lines) > 0 {
		titlePart := "─ " + title + " "
		remaining := width - len(titlePart) - 2
		if remaining < 0 {
			remaining = 0
		}
		newTop := "┌" + titlePart + strings.Repeat("─", remaining) + "┐"
		lines[0] = newTop
	}

	return strings.Join(lines, "\n")
}

// renderDataSections renders hosts, IPs, and paths sections
func (m Model) renderDataSections(layout *Layout, availableHeight int) string {
	// Calculate how many rows each section can have
	// Reserve lines for headers and borders
	sectionOverhead := 3 // title border + header row + bottom border

	var sections []string

	switch layout.DataColumns {
	case 1:
		// Stacked layout
		perSection := (availableHeight - sectionOverhead*3) / 3
		if perSection < 1 {
			perSection = 1
		}

		hostSection := m.renderHostsSectionBordered(m.width, perSection, m.section == SectionHosts)
		ipSection := m.renderIPsSectionBordered(m.width, perSection, m.section == SectionIPs)
		pathSection := m.renderPathsSectionBordered(m.width, perSection, false)

		sections = append(sections, hostSection, ipSection, pathSection)

	default:
		// Side by side (2 or 3 columns)
		perSection := (availableHeight - sectionOverhead*2) / 2
		if perSection < 1 {
			perSection = 1
		}

		// Hosts and IPs side by side
		colWidth := (m.width - 2) / 2
		hostSection := m.renderHostsSectionBordered(colWidth, perSection, m.section == SectionHosts)
		ipSection := m.renderIPsSectionBordered(colWidth, perSection, m.section == SectionIPs)

		sideBySide := m.joinSideBySide(hostSection, ipSection, colWidth)
		sections = append(sections, sideBySide)

		// Paths below
		pathSection := m.renderPathsSectionBordered(m.width, perSection, false)
		sections = append(sections, pathSection)
	}

	return strings.Join(sections, "\n")
}

// renderHostsSectionBordered renders hosts section with border
func (m Model) renderHostsSectionBordered(width, maxRows int, active bool) string {
	innerWidth := width - 4 // account for borders
	content := m.renderHostsContent(maxRows, innerWidth)
	title := fmt.Sprintf("Hosts (%d)", m.uniqueHosts)
	if m.filter.Host != "" {
		title = fmt.Sprintf("Host: %s", m.filter.Host)
	}
	return m.renderBorderedSection(title, content, width, active)
}

// renderIPsSectionBordered renders IPs section with border
func (m Model) renderIPsSectionBordered(width, maxRows int, active bool) string {
	innerWidth := width - 4 // account for borders
	content := m.renderIPsContent(maxRows, innerWidth)
	title := fmt.Sprintf("IPs (%d)", m.uniqueIPs)
	if m.filter.IP != "" {
		title = fmt.Sprintf("IP: %s", m.filter.IP)
	}
	return m.renderBorderedSection(title, content, width, active)
}

// renderPathsSectionBordered renders paths section with border
func (m Model) renderPathsSectionBordered(width, maxRows int, active bool) string {
	content := m.renderPathsContent(maxRows, width-4)
	title := fmt.Sprintf("Paths (%d)", m.uniquePaths)
	return m.renderBorderedSection(title, content, width, active)
}

// renderHostsContent renders hosts table content (no border)
func (m Model) renderHostsContent(maxRows, width int) string {
	return m.renderTableContent(m.topHosts, m.hostCursor, m.section == SectionHosts, m.filter.Host != "", m.hostErrRates, maxRows, width)
}

// renderIPsContent renders IPs table content (no border)
func (m Model) renderIPsContent(maxRows, width int) string {
	return m.renderTableContent(m.topIPs, m.ipCursor, m.section == SectionIPs, m.filter.IP != "", m.ipErrRates, maxRows, width)
}

// renderTableContent renders a data table with header row
func (m Model) renderTableContent(items []store.CountItem, cursor int, active, dimmed bool, errRates map[string]store.ErrorRates, maxRows, width int) string {
	// Calculate dynamic label length based on available width
	// Format: "  <label>  <count>  <pct>%  <4xx>  <5xx>"
	// Fixed parts: 2 (cursor) + 8 (count) + 7 (pct) + 6 (4xx) + 6 (5xx) + 4 (spacing) = 33 chars
	fixedWidth := 33
	maxLabelLen := width - fixedWidth
	if maxLabelLen < 15 {
		maxLabelLen = 15
	}
	if maxLabelLen > 60 {
		maxLabelLen = 60
	}

	var lines []string

	// Header row
	header := fmt.Sprintf("  %-*s %7s %6s %5s %5s",
		maxLabelLen, "Name", "Count", "%", "4xx", "5xx")
	lines = append(lines, tableHeaderStyle.Render(header))

	if len(items) == 0 {
		lines = append(lines, tableRowDimStyle.Render("  No data"))
		return strings.Join(lines, "\n")
	}

	// Limit items to maxRows (subtract 1 for header)
	displayItems := items
	displayMax := maxRows - 1
	if displayMax < 1 {
		displayMax = 1
	}
	if len(displayItems) > displayMax {
		displayItems = displayItems[:displayMax]
	}

	// Calculate total for percentages
	var total int64
	for _, item := range items {
		total += item.Count
	}

	for i, item := range displayItems {
		label := item.Label
		if len(label) > maxLabelLen {
			label = label[:maxLabelLen-3] + "..."
		}

		pct := float64(item.Count) * 100 / float64(max64(1, total))

		var rate4xx, rate5xx float64
		if rates, ok := errRates[item.Label]; ok {
			rate4xx = rates.Rate4xx
			rate5xx = rates.Rate5xx
		}

		isSelected := active && i == cursor

		rate4xxStr := "    -"
		rate5xxStr := "    -"
		if rate4xx > 0 {
			if dimmed || isSelected {
				rate4xxStr = fmt.Sprintf("%5.1f", rate4xx)
			} else {
				rate4xxStr = status4xxStyle.Render(fmt.Sprintf("%5.1f", rate4xx))
			}
		}
		if rate5xx > 0 {
			if dimmed || isSelected {
				rate5xxStr = fmt.Sprintf("%5.1f", rate5xx)
			} else {
				rate5xxStr = status5xxStyle.Render(fmt.Sprintf("%5.1f", rate5xx))
			}
		}

		line := fmt.Sprintf("%-*s %7s %5.1f%% %s %s",
			maxLabelLen, label, formatNumber(item.Count), pct, rate4xxStr, rate5xxStr)

		var style lipgloss.Style
		if dimmed {
			style = tableRowDimStyle
		} else if isSelected {
			line = "> " + line
			style = tableRowSelectedStyle
		} else {
			line = "  " + line
			style = tableRowStyle
		}

		lines = append(lines, style.Render(line))
	}

	return strings.Join(lines, "\n")
}

// renderPathsContent renders paths table content
func (m Model) renderPathsContent(maxRows, width int) string {
	// Calculate max path length dynamically
	// Format: "  <path>  <count>  <pct>%  <4xx>  <5xx>"
	// Fixed parts: 2 (indent) + 8 (count) + 7 (pct) + 6 (4xx) + 6 (5xx) + 4 (spacing) = 33 chars
	fixedWidth := 33
	maxPathLen := width - fixedWidth
	if maxPathLen < 15 {
		maxPathLen = 15
	}
	if maxPathLen > 80 {
		maxPathLen = 80
	}

	var lines []string

	// Header row
	header := fmt.Sprintf("  %-*s %7s %6s %5s %5s",
		maxPathLen, "Path", "Count", "%", "4xx", "5xx")
	lines = append(lines, tableHeaderStyle.Render(header))

	if len(m.topPaths) == 0 {
		lines = append(lines, tableRowDimStyle.Render("  No data"))
		return strings.Join(lines, "\n")
	}

	// Limit items (subtract 1 for header)
	displayItems := m.topPaths
	displayMax := maxRows - 1
	if displayMax < 1 {
		displayMax = 1
	}
	if len(displayItems) > displayMax {
		displayItems = displayItems[:displayMax]
	}

	var total int64
	for _, item := range m.topPaths {
		total += item.Count
	}

	for _, item := range displayItems {
		label := item.Label
		if len(label) > maxPathLen {
			label = label[:maxPathLen-3] + "..."
		}

		pct := float64(item.Count) * 100 / float64(max64(1, total))

		var rate4xx, rate5xx float64
		if rates, ok := m.pathErrRates[item.Label]; ok {
			rate4xx = rates.Rate4xx
			rate5xx = rates.Rate5xx
		}

		rate4xxStr := "    -"
		rate5xxStr := "    -"
		if rate4xx > 0 {
			rate4xxStr = status4xxStyle.Render(fmt.Sprintf("%5.1f", rate4xx))
		}
		if rate5xx > 0 {
			rate5xxStr = status5xxStyle.Render(fmt.Sprintf("%5.1f", rate5xx))
		}

		line := fmt.Sprintf("  %-*s %7s %5.1f%% %s %s",
			maxPathLen, label, formatNumber(item.Count), pct, rate4xxStr, rate5xxStr)
		lines = append(lines, tableRowStyle.Render(line))
	}

	return strings.Join(lines, "\n")
}

// joinSideBySide joins two sections horizontally
func (m Model) joinSideBySide(left, right string, colWidth int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	var result []string
	for i := 0; i < maxLines; i++ {
		var leftLine, rightLine string
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}

		leftLine = padRight(leftLine, colWidth)
		result = append(result, leftLine+rightLine)
	}

	return strings.Join(result, "\n")
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

const noDataWarningThreshold = 30 * time.Second

func (m Model) renderHeader() string {
	elapsed := time.Since(m.startTime).Round(time.Second)

	// Build header with current rate instead of lifetime average
	header := fmt.Sprintf("hstat | %s | %s reqs | %.1f/s",
		elapsed,
		formatNumber(m.stats.TotalCount),
		m.currentRate,
	)

	result := headerStyle.Render(header)

	// Error rates with trend
	if m.stats.TotalCount > 0 {
		errPart := ""
		if m.rate4xx > 0 {
			errPart += status4xxStyle.Render(fmt.Sprintf("4xx:%.1f%%", m.rate4xx))
		}
		if m.rate5xx > 0 {
			if errPart != "" {
				errPart += " "
			}
			errPart += status5xxStyle.Render(fmt.Sprintf("5xx:%.1f%%", m.rate5xx))
		}

		// 1m trend
		switch m.trend {
		case store.TrendUp:
			errPart += " " + trendUpStyle.Render("1m↑")
		case store.TrendDown:
			errPart += " " + trendDownStyle.Render("1m↓")
		}
		// 5m trend
		switch m.trend5m {
		case store.TrendUp:
			errPart += " " + trendUpStyle.Render("5m↑")
		case store.TrendDown:
			errPart += " " + trendDownStyle.Render("5m↓")
		}

		if errPart != "" {
			result += "  " + errPart
		}
	}

	// Stream status warnings
	if m.streamEnded {
		result += "  " + streamEndedStyle.Render("⚠ STREAM ENDED")
	} else if !m.lastEntryTime.IsZero() {
		sinceLastEntry := time.Since(m.lastEntryTime)
		if sinceLastEntry > noDataWarningThreshold {
			secs := int(sinceLastEntry.Seconds())
			result += "  " + warningStyle.Render(fmt.Sprintf("⚠ no data for %ds", secs))
		}
	}

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
	return m.renderTableWithErrors("Host", m.uniqueHosts, m.topHosts, m.otherHosts, m.hostCursor, active, m.filter.Host != "", m.hostErrRates)
}

func (m Model) renderIPs() string {
	active := m.section == SectionIPs
	// Dim IPs when filtering BY IP (IP is the filter source)
	return m.renderTableWithErrors("IP", m.uniqueIPs, m.topIPs, m.otherIPs, m.ipCursor, active, m.filter.IP != "", m.ipErrRates)
}

func (m Model) renderPaths() string {
	var b strings.Builder

	// Calculate max path length based on terminal width
	// Format: "  <path>  <count>  <pct>%  <4xx>  <5xx>"
	// Fixed parts: 2 (indent) + 10 (count) + 7 (pct) + 6 (4xx) + 6 (5xx) = 31 chars
	fixedWidth := 31
	maxPathLen := m.width - fixedWidth - 10 // extra padding
	if maxPathLen < 20 {
		maxPathLen = 20
	}
	if maxPathLen > 60 {
		maxPathLen = 60
	}

	// Build header
	header := fmt.Sprintf("  %-*s  %8s  %5s  %5s  %5s",
		maxPathLen,
		fmt.Sprintf("Path (%d)", m.uniquePaths),
		"Count", "%", "4xx", "5xx")

	b.WriteString(tableHeaderStyle.Render(header))
	b.WriteString("\n")

	if len(m.topPaths) == 0 {
		b.WriteString(tableRowDimStyle.Render("  No data"))
		return b.String()
	}

	// Calculate total for percentages
	var total int64
	for _, item := range m.topPaths {
		total += item.Count
	}

	for _, item := range m.topPaths {
		label := item.Label
		if len(label) > maxPathLen {
			label = label[:maxPathLen-3] + "..."
		}

		pct := float64(item.Count) * 100 / float64(max64(1, total))

		// Get error rates
		var rate4xx, rate5xx float64
		if rates, ok := m.pathErrRates[item.Label]; ok {
			rate4xx = rates.Rate4xx
			rate5xx = rates.Rate5xx
		}

		// Format error rates
		rate4xxStr := "    -"
		rate5xxStr := "    -"
		if rate4xx > 0 {
			rate4xxStr = status4xxStyle.Render(fmt.Sprintf("%5.1f", rate4xx))
		}
		if rate5xx > 0 {
			rate5xxStr = status5xxStyle.Render(fmt.Sprintf("%5.1f", rate5xx))
		}

		line := fmt.Sprintf("  %-*s  %8s  %5.1f  %s  %s",
			maxPathLen, label, formatNumber(item.Count), pct, rate4xxStr, rate5xxStr)
		b.WriteString(tableRowStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderTableWithErrors(columnName string, uniqueCount int, items []store.CountItem, other int64, cursor int, active bool, dimmed bool, errRates map[string]store.ErrorRates) string {
	var b strings.Builder

	maxLabelLen := 22

	// Header row with column names
	headerStyle := tableHeaderStyle
	if active {
		headerStyle = sectionTitleActiveStyle
	}

	// Build header: "  Host (42)              Count     %    4xx    5xx"
	header := fmt.Sprintf("  %-*s  %8s  %5s  %5s  %5s",
		maxLabelLen,
		fmt.Sprintf("%s (%d)", columnName, uniqueCount),
		"Count", "%", "4xx", "5xx")

	b.WriteString(headerStyle.Render(header))
	if active {
		b.WriteString(" " + helpStyle.Render("[j/k]"))
	}
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString(tableRowDimStyle.Render("  No data"))
		return b.String()
	}

	// Calculate total for percentages
	var total int64
	for _, item := range items {
		total += item.Count
	}
	total += other

	for i, item := range items {
		label := item.Label
		if len(label) > maxLabelLen {
			label = label[:maxLabelLen-3] + "..."
		}

		pct := float64(item.Count) * 100 / float64(max64(1, total))

		// Get error rates
		var rate4xx, rate5xx float64
		if rates, ok := errRates[item.Label]; ok {
			rate4xx = rates.Rate4xx
			rate5xx = rates.Rate5xx
		}

		// Determine if this row needs special styling (which prevents nested ANSI)
		isSelected := active && i == cursor

		// Build the line - for selected rows, don't use colored error rates
		var line string
		if dimmed || isSelected {
			// Plain text for dimmed or selected rows
			rate4xxStr := "    -"
			rate5xxStr := "    -"
			if rate4xx > 0 {
				rate4xxStr = fmt.Sprintf("%5.1f", rate4xx)
			}
			if rate5xx > 0 {
				rate5xxStr = fmt.Sprintf("%5.1f", rate5xx)
			}
			line = fmt.Sprintf("%-*s  %8s  %5.1f  %s  %s",
				maxLabelLen, label, formatNumber(item.Count), pct, rate4xxStr, rate5xxStr)
		} else {
			// Colored error rates for normal rows
			rate4xxStr := "    -"
			rate5xxStr := "    -"
			if rate4xx > 0 {
				rate4xxStr = status4xxStyle.Render(fmt.Sprintf("%5.1f", rate4xx))
			}
			if rate5xx > 0 {
				rate5xxStr = status5xxStyle.Render(fmt.Sprintf("%5.1f", rate5xx))
			}
			line = fmt.Sprintf("%-*s  %8s  %5.1f  %s  %s",
				maxLabelLen, label, formatNumber(item.Count), pct, rate4xxStr, rate5xxStr)
		}

		var style lipgloss.Style
		if dimmed {
			line = "  " + line
			style = tableRowDimStyle
		} else if isSelected {
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
		line := fmt.Sprintf("  %-*s  %8s  %5.1f", maxLabelLen, "(other)", formatNumber(other), pct)
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

// helpContent returns the help text for the modal
func helpContent() string {
	return `Navigation:
  Tab / l        Next section
  Shift+Tab / h  Previous section
  j / Down       Move cursor down
  k / Up         Move cursor up
  g              Jump to top
  G              Jump to bottom

Actions:
  Enter          Filter by selected host/IP
  w              Whois lookup (when IP selected)
  i              ipinfo.io lookup (when IP selected)
  Esc            Clear filter (or close modal)
  q / Ctrl+C     Quit`
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

	// Truncate long lines safely (by visible width, not byte length)
	for i, line := range lines {
		visibleWidth := lipgloss.Width(line)
		if visibleWidth > modalWidth-6 {
			// Truncate by runes to avoid breaking multi-byte characters
			runes := []rune(line)
			targetLen := modalWidth - 9
			if targetLen > 0 && targetLen < len(runes) {
				lines[i] = string(runes[:targetLen]) + "..."
			}
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
			// Modal line - use clean padding, don't try to show dimmed background on sides
			modalLineIdx := i - startY
			modalLine := modalLines[modalLineIdx]

			// Pad left side with spaces, then modal, then fill to terminal width
			prefix := strings.Repeat(" ", startX)
			result[i] = prefix + modalLine
		} else {
			// Non-modal line - show dimmed background
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
