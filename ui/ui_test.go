package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/betternow/hstat/parser"
	"github.com/betternow/hstat/store"
	tea "github.com/charmbracelet/bubbletea"
)

// testEntry creates a parser.Entry for testing
func testEntry(status int, host, ip string) *parser.Entry {
	return &parser.Entry{
		Timestamp: time.Now(),
		Status:    status,
		Service:   10,
		Connect:   1,
		Host:      host,
		IP:        ip,
	}
}

func TestNewModel(t *testing.T) {
	s := store.New(5 * time.Minute)
	m := NewModel(s, 15, time.Second)

	if m.topN != 15 {
		t.Errorf("expected topN 15, got %d", m.topN)
	}
	if m.refreshRate != time.Second {
		t.Errorf("expected refreshRate 1s, got %v", m.refreshRate)
	}
	if m.section != SectionHosts {
		t.Errorf("expected section SectionHosts, got %v", m.section)
	}
	if m.showHelp {
		t.Error("expected showHelp false")
	}
	if m.modal.Visible {
		t.Error("expected modal not visible")
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n        int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{10000, "10.0k"},
		{999999, "1000.0k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{10000000, "10.0M"},
	}

	for _, tc := range tests {
		result := formatNumber(tc.n)
		if result != tc.expected {
			t.Errorf("formatNumber(%d) = %s, expected %s", tc.n, result, tc.expected)
		}
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"\x1b[38;5;196mcolor\x1b[0m", "color"},
		{"no codes here", "no codes here"},
		{"", ""},
	}

	for _, tc := range tests {
		result := stripAnsi(tc.input)
		if result != tc.expected {
			t.Errorf("stripAnsi(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestSubstring(t *testing.T) {
	tests := []struct {
		input    string
		start    int
		end      int
		expected string
	}{
		{"hello", 0, 5, "hello"},
		{"hello", 0, 3, "hel"},
		{"hello", 2, 5, "llo"},
		{"hello", 0, 10, "hello"}, // end beyond length
		{"hello", 10, 15, ""},     // start beyond length
		{"", 0, 5, ""},
	}

	for _, tc := range tests {
		result := substring(tc.input, tc.start, tc.end)
		if result != tc.expected {
			t.Errorf("substring(%q, %d, %d) = %q, expected %q", tc.input, tc.start, tc.end, result, tc.expected)
		}
	}
}

func TestMax64(t *testing.T) {
	tests := []struct {
		a, b     int64
		expected int64
	}{
		{1, 2, 2},
		{2, 1, 2},
		{0, 0, 0},
		{-1, 1, 1},
		{100, 100, 100},
	}

	for _, tc := range tests {
		result := max64(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("max64(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{0, 0, 0},
		{-1, 1, -1},
		{100, 100, 100},
	}

	for _, tc := range tests {
		result := min(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected int // expected length
	}{
		{"hello", 10, 10},
		{"hello", 5, 5},
		{"hello", 3, 5}, // don't truncate
		{"", 5, 5},
	}

	for _, tc := range tests {
		result := padRight(tc.input, tc.width)
		// Check that visible width is as expected
		if len(result) < tc.expected {
			t.Errorf("padRight(%q, %d) length = %d, expected >= %d", tc.input, tc.width, len(result), tc.expected)
		}
	}
}

func TestHandleKey_Quit(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)

	// Test 'q' key
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command for key q")
	}
}

func TestHandleKey_Help(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)

	// Toggle help on
	newM, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model := newM.(Model)
	if !model.showHelp {
		t.Error("expected showHelp true after pressing ?")
	}

	// Toggle help off
	newM, _ = model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model = newM.(Model)
	if model.showHelp {
		t.Error("expected showHelp false after pressing ? again")
	}
}

func TestHandleKey_SectionNavigation(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50

	// Start at hosts
	if m.section != SectionHosts {
		t.Error("expected to start at SectionHosts")
	}

	// Tab to IPs
	newM, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	model := newM.(Model)
	if model.section != SectionIPs {
		t.Error("expected SectionIPs after Tab")
	}

	// Tab back to hosts (wraps)
	newM, _ = model.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	model = newM.(Model)
	if model.section != SectionHosts {
		t.Error("expected SectionHosts after Tab (wrap)")
	}
}

func TestHandleKey_CursorMovement(t *testing.T) {
	s := store.New(0)
	// Add some test data
	for i := 0; i < 5; i++ {
		s.Add(testEntry(200, "host.com", "1.1.1.1"))
	}

	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.refreshData()

	// Move cursor down
	m.moveCursor(1)
	if m.hostCursor != 0 { // Only one host, so cursor stays at 0
		t.Errorf("expected hostCursor 0, got %d", m.hostCursor)
	}
}

func TestHandleKey_Filter(t *testing.T) {
	s := store.New(0)
	s.Add(testEntry(200, "api.com", "1.1.1.1"))
	s.Add(testEntry(200, "web.com", "2.2.2.2"))

	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.refreshData()

	// Apply filter
	m.applyFilter()
	if m.filter.Host == "" {
		t.Error("expected filter to be applied")
	}

	// Clear filter with Esc
	newM, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	model := newM.(Model)
	if model.filter.Host != "" || model.filter.IP != "" {
		t.Error("expected filter to be cleared")
	}
}

func TestHandleKey_ModalDismissal(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.modal.Visible = true
	m.modal.Content = "test content"

	dismissKeys := []tea.KeyType{tea.KeyEsc, tea.KeyEnter}
	for _, keyType := range dismissKeys {
		m.modal.Visible = true
		newM, _ := m.handleKey(tea.KeyMsg{Type: keyType})
		model := newM.(Model)
		if model.modal.Visible {
			t.Errorf("expected modal to be dismissed with key type %v", keyType)
		}
	}
}

func TestView_MinimumSize(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)

	// Zero size
	m.width = 0
	m.height = 0
	view := m.View()
	if !strings.Contains(view, "Initializing") {
		t.Error("expected Initializing message for zero size")
	}

	// Too small
	m.width = 30
	m.height = 10
	view = m.View()
	if !strings.Contains(view, "too small") {
		t.Error("expected 'too small' message")
	}
}

func TestView_Help(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.showHelp = true

	view := m.View()
	if !strings.Contains(view, "Navigation") {
		t.Error("expected help to contain Navigation section")
	}
	if !strings.Contains(view, "Whois") {
		t.Error("expected help to mention Whois")
	}
	if !strings.Contains(view, "ipinfo") {
		t.Error("expected help to mention ipinfo")
	}
}

func TestRenderWithModal_FixedPosition(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 30
	m.modal.Visible = true
	m.modal.Title = "Test Modal"
	m.modal.Content = "Test content"

	// Render with short background
	shortBg := "line1\nline2\nline3"
	result1 := m.renderWithModal(shortBg)
	lines1 := strings.Split(result1, "\n")

	// Render with longer background
	longBg := strings.Repeat("line\n", 20)
	result2 := m.renderWithModal(longBg)
	lines2 := strings.Split(result2, "\n")

	// Both should have the same number of lines (m.height)
	if len(lines1) != m.height {
		t.Errorf("expected %d lines, got %d", m.height, len(lines1))
	}
	if len(lines2) != m.height {
		t.Errorf("expected %d lines, got %d", m.height, len(lines2))
	}

	// Modal should contain the title
	fullResult := result1
	if !strings.Contains(fullResult, "Test Modal") {
		t.Error("expected modal to contain title")
	}
}

func TestStatusStyle(t *testing.T) {
	// Just verify it doesn't panic for various status codes
	statusCodes := []int{100, 200, 201, 301, 302, 400, 404, 500, 503}
	for _, code := range statusCodes {
		style := StatusStyle(code)
		// Style should render without panic
		_ = style.Render("test")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)

	newM, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := newM.(Model)

	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
}

func TestStreamEndedMsg(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)

	newM, _ := m.Update(StreamEndedMsg{})
	model := newM.(Model)

	if !model.streamEnded {
		t.Error("expected streamEnded to be true")
	}
}

func TestWhoisResultMsg(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.modal.Visible = true
	m.modal.Loading = true

	// Success
	newM, _ := m.Update(WhoisResultMsg{IP: "1.2.3.4", Content: "Whois data"})
	model := newM.(Model)
	if model.modal.Loading {
		t.Error("expected loading to be false")
	}
	if model.modal.Content != "Whois data" {
		t.Errorf("expected content 'Whois data', got %q", model.modal.Content)
	}

	// Error
	m.modal.Loading = true
	newM, _ = m.Update(WhoisResultMsg{IP: "1.2.3.4", Err: errTest})
	model = newM.(Model)
	if !strings.Contains(model.modal.Content, "Error") {
		t.Error("expected error message in content")
	}
}

func TestIpinfoResultMsg(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.modal.Visible = true
	m.modal.Loading = true

	newM, _ := m.Update(IpinfoResultMsg{IP: "1.2.3.4", Content: "IP info"})
	model := newM.(Model)
	if model.modal.Loading {
		t.Error("expected loading to be false")
	}
	if model.modal.Content != "IP info" {
		t.Errorf("expected content 'IP info', got %q", model.modal.Content)
	}
}

func TestEntryMsg_UpdatesLastEntryTime(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)

	// Initially zero
	if !m.lastEntryTime.IsZero() {
		t.Error("expected lastEntryTime to be zero initially")
	}

	before := time.Now()
	newM, _ := m.Update(EntryMsg{Entry: testEntry(200, "test.com", "1.1.1.1")})
	after := time.Now()

	model := newM.(Model)
	if model.lastEntryTime.Before(before) || model.lastEntryTime.After(after) {
		t.Errorf("expected lastEntryTime between %v and %v, got %v", before, after, model.lastEntryTime)
	}
}

func TestRenderHeader_ShowsNoDataWarning(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50

	// Set lastEntryTime to 45 seconds ago
	m.lastEntryTime = time.Now().Add(-45 * time.Second)

	header := m.renderHeader()
	if !strings.Contains(header, "no data") {
		t.Errorf("expected 'no data' warning in header when no entries for 45s, got: %s", header)
	}
}

func TestRenderHeader_NoWarningWhenRecentData(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50

	// Set lastEntryTime to 5 seconds ago
	m.lastEntryTime = time.Now().Add(-5 * time.Second)

	header := m.renderHeader()
	if strings.Contains(header, "no data") {
		t.Errorf("expected no warning when data is recent, got: %s", header)
	}
}

func TestRenderHeader_NoWarningWhenNoDataYet(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50

	// lastEntryTime is zero (no data received yet)
	header := m.renderHeader()
	if strings.Contains(header, "no data") {
		t.Errorf("expected no warning when no data received yet (still initializing), got: %s", header)
	}
}

func TestRenderHeader_NoDataWarningNotShownWhenStreamEnded(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.streamEnded = true
	m.lastEntryTime = time.Now().Add(-45 * time.Second)

	header := m.renderHeader()
	// When stream has ended, we show "stream ended" not "no data"
	if strings.Contains(header, "no data") {
		t.Errorf("expected no 'no data' warning when stream ended, got: %s", header)
	}
	if !strings.Contains(header, "STREAM ENDED") {
		t.Errorf("expected 'STREAM ENDED' in header, got: %s", header)
	}
}

func TestRenderHeader_StreamEndedIsProminent(t *testing.T) {
	s := store.New(0)
	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.streamEnded = true

	header := m.renderHeader()
	// Should contain STREAM ENDED in uppercase to be prominent
	if !strings.Contains(header, "STREAM ENDED") {
		t.Errorf("expected prominent 'STREAM ENDED' in header, got: %s", header)
	}
}

func TestPathsShownWhenFilteredByHost(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/orders", IP: "1.1.1.1"})

	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.filter.Host = "api.com"
	m.refreshData()

	// Should have paths cached
	if len(m.topPaths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(m.topPaths))
	}

	// View should contain path information
	view := m.View()
	if !strings.Contains(view, "/users") {
		t.Error("expected view to contain /users path")
	}
	if !strings.Contains(view, "/orders") {
		t.Error("expected view to contain /orders path")
	}
}

func TestPathsNotShownWithoutHostFilter(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})

	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.refreshData()

	// Should have no paths cached without host filter
	if len(m.topPaths) != 0 {
		t.Errorf("expected 0 paths without host filter, got %d", len(m.topPaths))
	}
}

func TestPathsSection_Title(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})

	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.filter.Host = "api.com"
	m.refreshData()

	view := m.View()
	if !strings.Contains(view, "Paths") {
		t.Error("expected view to contain 'Paths' section title when filtered by host")
	}
}

func TestPathsShownWhenFilteredByIP(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "web.com", Path: "/orders", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/admin", IP: "2.2.2.2"})

	m := NewModel(s, 15, time.Second)
	m.width = 100
	m.height = 50
	m.filter.IP = "1.1.1.1"
	m.refreshData()

	// Should have paths cached for this IP
	if len(m.topPaths) != 2 {
		t.Errorf("expected 2 paths for IP 1.1.1.1, got %d", len(m.topPaths))
	}

	// View should contain path information
	view := m.View()
	if !strings.Contains(view, "/users") {
		t.Error("expected view to contain /users path")
	}
	if !strings.Contains(view, "/orders") {
		t.Error("expected view to contain /orders path")
	}
	if strings.Contains(view, "/admin") {
		t.Error("expected view NOT to contain /admin path (belongs to different IP)")
	}
}

func TestRenderPaths_CountsBeforePath(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})

	m := NewModel(s, 15, time.Second)
	m.width = 80
	m.height = 50
	m.filter.Host = "api.com"
	m.refreshData()

	paths := m.renderPaths()

	// Find the line containing /users
	lines := strings.Split(paths, "\n")
	var pathLine string
	for _, line := range lines {
		if strings.Contains(line, "/users") {
			pathLine = line
			break
		}
	}

	if pathLine == "" {
		t.Fatal("expected to find line with /users")
	}

	// Count/percentage should appear before the path
	// The line format should be: "  <count>  <pct>%  <path>"
	stripped := stripAnsi(pathLine)
	countIdx := strings.Index(stripped, "2") // count of 2
	pathIdx := strings.Index(stripped, "/users")

	if countIdx == -1 || pathIdx == -1 {
		t.Fatalf("expected to find count and path in line: %q", stripped)
	}

	if countIdx > pathIdx {
		t.Errorf("expected count before path, but count at %d, path at %d in: %q", countIdx, pathIdx, stripped)
	}
}

func TestRenderPaths_WideTerminalExpandsPath(t *testing.T) {
	s := store.New(0)
	// Path long enough to be truncated at 60 chars but fit at 140 chars
	longPath := "/api/v2/users/12345678/orders/87654321/items/details/extended/view"
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: longPath, IP: "1.1.1.1"})

	m := NewModel(s, 15, time.Second)
	m.filter.Host = "api.com"

	// Narrow terminal (width 80 - 20 fixed = 60 max path)
	m.width = 80
	m.height = 50
	m.refreshData()
	narrowPaths := m.renderPaths()
	narrowStripped := stripAnsi(narrowPaths)

	// Wide terminal (width 160 - 20 fixed = 140 max path)
	m.width = 160
	m.refreshData()
	widePaths := m.renderPaths()
	wideStripped := stripAnsi(widePaths)

	// Count how much of the path is visible in each
	narrowPathVisible := countPathChars(narrowStripped, longPath)
	widePathVisible := countPathChars(wideStripped, longPath)

	// Wide terminal should show more of the path (full path at 68 chars)
	if widePathVisible <= narrowPathVisible {
		t.Errorf("expected wide terminal to show more path chars (%d) than narrow (%d)",
			widePathVisible, narrowPathVisible)
	}
}

// countPathChars returns how many characters of the path are visible in the output
func countPathChars(output, path string) int {
	// Check if full path is present
	if strings.Contains(output, path) {
		return len(path)
	}
	// Otherwise check for truncated version
	for i := len(path) - 1; i > 0; i-- {
		prefix := path[:i]
		if strings.Contains(output, prefix) {
			return i
		}
	}
	return 0
}

// Test error for error case
var errTest = testError{}

type testError struct{}

func (testError) Error() string { return "test error" }
