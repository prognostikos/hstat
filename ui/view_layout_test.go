package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/betternow/hstat/parser"
	"github.com/betternow/hstat/store"
)

func TestView_FitsWithinTerminalHeight(t *testing.T) {
	s := store.New(0)

	// Add lots of data to potentially overflow
	for i := 0; i < 100; i++ {
		s.Add(&parser.Entry{
			Status: 200 + (i % 5),
			Host:   "host" + string(rune('a'+i%26)) + ".com",
			Path:   "/path" + string(rune('0'+i%10)),
			IP:     "1.1.1." + string(rune('0'+i%10)),
		})
	}

	m := NewModel(s, time.Second)
	m.width = 120
	m.height = 30 // Small terminal
	m.refreshData()

	view := m.View()
	lines := strings.Split(view, "\n")

	// View should never exceed terminal height
	if len(lines) > m.height {
		t.Errorf("View has %d lines but terminal height is %d - content overflows", len(lines), m.height)
	}
}

func TestView_FitsWithinTerminalHeight_VerySmall(t *testing.T) {
	s := store.New(0)

	// Add data
	for i := 0; i < 50; i++ {
		s.Add(&parser.Entry{
			Status: 200,
			Host:   "example.com",
			Path:   "/api",
			IP:     "1.2.3.4",
		})
	}

	m := NewModel(s, time.Second)
	m.width = 80
	m.height = 20 // Minimum height
	m.refreshData()

	view := m.View()
	lines := strings.Split(view, "\n")

	if len(lines) > m.height {
		t.Errorf("View has %d lines but terminal height is %d - content overflows at minimum size", len(lines), m.height)
	}
}

func TestView_StatusCodesInColumns(t *testing.T) {
	s := store.New(0)

	// Add various status codes
	for i := 0; i < 10; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	}
	for i := 0; i < 5; i++ {
		s.Add(&parser.Entry{Status: 404, Host: "a.com", IP: "1.1.1.1"})
	}
	for i := 0; i < 3; i++ {
		s.Add(&parser.Entry{Status: 500, Host: "a.com", IP: "1.1.1.1"})
	}

	m := NewModel(s, time.Second)
	m.width = 120 // Wide enough for columns
	m.height = 40
	m.refreshData()

	view := m.View()

	// Status codes should appear on the same line (columnar layout)
	// Look for "2xx" and "4xx" on the same line
	lines := strings.Split(view, "\n")
	foundColumnar := false
	for _, line := range lines {
		if strings.Contains(line, "2xx") && strings.Contains(line, "4xx") {
			foundColumnar = true
			break
		}
	}

	if !foundColumnar {
		t.Error("expected status code categories (2xx, 4xx) to appear on same line in columnar layout")
	}
}

func TestView_SectionsHaveBorders(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "example.com", Path: "/api", IP: "1.2.3.4"})

	m := NewModel(s, time.Second)
	m.width = 120
	m.height = 40
	m.refreshData()

	view := m.View()

	// Should contain border characters (sharp corners: ┌ ┐ └ ┘ and lines: ─ │)
	if !strings.Contains(view, "┌") {
		t.Error("expected view to contain top-left corner border character ┌")
	}
	if !strings.Contains(view, "┘") {
		t.Error("expected view to contain bottom-right corner border character ┘")
	}
	if !strings.Contains(view, "─") {
		t.Error("expected view to contain horizontal border character ─")
	}
	if !strings.Contains(view, "│") {
		t.Error("expected view to contain vertical border character │")
	}
}

func TestView_HeaderSectionHasBorder(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "example.com", IP: "1.2.3.4"})

	m := NewModel(s, time.Second)
	m.width = 120
	m.height = 40
	m.refreshData()

	view := m.View()
	lines := strings.Split(view, "\n")

	// First line should be a border (starts with ┌)
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "┌") {
		t.Error("expected first line to be a border starting with ┌")
	}
}

func TestView_ActiveSectionHighlighted(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "example.com", IP: "1.2.3.4"})

	m := NewModel(s, time.Second)
	m.width = 120
	m.height = 40
	m.section = SectionHosts
	m.refreshData()

	view := m.View()

	// The active section (Hosts) should be visible
	// We can't easily test for magenta color, but we can verify the section exists
	if !strings.Contains(view, "Host") {
		t.Error("expected view to contain Hosts section")
	}
}

func TestView_DataSectionsLimitedByHeight(t *testing.T) {
	s := store.New(0)

	// Add many unique hosts
	for i := 0; i < 50; i++ {
		host := "host" + string(rune('a'+i%26)) + string(rune('0'+i/26)) + ".com"
		s.Add(&parser.Entry{Status: 200, Host: host, IP: "1.1.1.1", Path: "/test"})
	}

	m := NewModel(s, time.Second)
	m.width = 120
	m.height = 25 // Limited height
	m.refreshData()

	view := m.View()
	lines := strings.Split(view, "\n")

	// Count how many hosts are visible
	hostCount := 0
	for _, line := range lines {
		if strings.Contains(line, ".com") && !strings.Contains(line, "hstat") {
			hostCount++
		}
	}

	// With limited height, we shouldn't show all 50 hosts
	// (exact number depends on layout, but should be much less than 50)
	if hostCount > 20 {
		t.Errorf("expected limited hosts due to height constraint, but found %d host lines", hostCount)
	}
}

func TestView_ColumnHeadersPresent(t *testing.T) {
	s := store.New(0)
	s.Add(&parser.Entry{Status: 200, Host: "example.com", Path: "/api", IP: "1.2.3.4"})

	m := NewModel(s, time.Second)
	m.width = 120
	m.height = 40
	m.refreshData()

	view := m.View()

	// Should have column headers for Count, %, 4xx, 5xx
	if !strings.Contains(view, "Count") {
		t.Error("expected view to contain 'Count' column header")
	}
	if !strings.Contains(view, "4xx") {
		t.Error("expected view to contain '4xx' column header")
	}
	if !strings.Contains(view, "5xx") {
		t.Error("expected view to contain '5xx' column header")
	}
}

func TestView_DynamicHostnameTruncation(t *testing.T) {
	s := store.New(0)
	longHost := "this-is-a-very-long-hostname-that-should-be-truncated.example.com"
	s.Add(&parser.Entry{Status: 200, Host: longHost, IP: "1.2.3.4"})

	// Wide terminal - should show more of the hostname
	m := NewModel(s, time.Second)
	m.width = 160
	m.height = 40
	m.refreshData()

	wideView := m.View()

	// Narrow terminal - should truncate more aggressively
	m.width = 80
	narrowView := m.View()

	// In wide view, should see more of the hostname than in narrow view
	// Count how much of the long hostname is visible
	wideVisible := 0
	narrowVisible := 0

	for i := 10; i <= len(longHost); i++ {
		if strings.Contains(wideView, longHost[:i]) {
			wideVisible = i
		}
		if strings.Contains(narrowView, longHost[:i]) {
			narrowVisible = i
		}
	}

	if wideVisible <= narrowVisible {
		t.Errorf("expected wide terminal to show more of hostname (wide: %d chars, narrow: %d chars)", wideVisible, narrowVisible)
	}
}
