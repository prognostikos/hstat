package ui

import (
	"time"

	"github.com/betternow/hstat/parser"
	"github.com/betternow/hstat/store"
	tea "github.com/charmbracelet/bubbletea"
)

// Section represents a navigable section
type Section int

const (
	SectionHosts Section = iota
	SectionIPs
)

// Filter represents the current filter state
type Filter struct {
	Host string
	IP   string
}

// Modal represents the current modal state
type Modal struct {
	Visible bool
	Title   string
	Content string
	Loading bool
}

// Model is the bubbletea model
type Model struct {
	store       *store.Store
	startTime   time.Time
	topN        int
	refreshRate time.Duration

	// UI state
	width         int
	height        int
	section       Section
	hostCursor    int
	ipCursor      int
	filter        Filter
	showHelp      bool
	streamEnded   bool
	lastEntryTime time.Time
	modal         Modal

	// Cached data for rendering
	stats        store.Stats
	statusCounts []store.StatusCountItem
	topHosts     []store.CountItem
	topIPs       []store.CountItem
	topPaths     []store.CountItem
	otherHosts   int64
	otherIPs     int64
}

// NewModel creates a new Model
func NewModel(s *store.Store, topN int, refreshRate time.Duration) Model {
	return Model{
		store:       s,
		startTime:   time.Now(),
		topN:        topN,
		refreshRate: refreshRate,
		section:     SectionHosts,
	}
}

// EntryMsg is sent when a new log entry is parsed
type EntryMsg struct {
	Entry *parser.Entry
}

// TickMsg is sent on each refresh tick
type TickMsg time.Time

// StreamEndedMsg is sent when stdin closes
type StreamEndedMsg struct{}

// WhoisResultMsg is sent when whois lookup completes
type WhoisResultMsg struct {
	IP      string
	Content string
	Err     error
}

// IpinfoResultMsg is sent when ipinfo.io lookup completes
type IpinfoResultMsg struct {
	IP      string
	Content string
	Err     error
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(m.refreshRate),
	)
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// refreshData updates cached data from the store
func (m *Model) refreshData() {
	m.store.Prune()
	m.stats = m.store.GetStats()
	m.statusCounts = m.store.GetStatusCounts(m.filter.Host, m.filter.IP)
	m.topHosts = m.store.GetTopHosts(m.topN, m.filter.IP)
	m.topIPs = m.store.GetTopIPs(m.topN, m.filter.Host)

	// Get paths when filtering by host or IP
	if m.filter.Host != "" || m.filter.IP != "" {
		m.topPaths = m.store.GetTopPaths(m.topN, m.filter.Host, m.filter.IP)
	} else {
		m.topPaths = nil
	}

	// Calculate "other" counts
	if m.filter.IP == "" {
		m.otherHosts = m.store.GetOtherCount(m.store.HostCounts, m.topHosts)
	} else {
		// When filtered, we don't show "other"
		m.otherHosts = 0
	}

	if m.filter.Host == "" {
		m.otherIPs = m.store.GetOtherCount(m.store.IPCounts, m.topIPs)
	} else {
		m.otherIPs = 0
	}

	// Clamp cursors
	if m.hostCursor >= len(m.topHosts) {
		m.hostCursor = max(0, len(m.topHosts)-1)
	}
	if m.ipCursor >= len(m.topIPs) {
		m.ipCursor = max(0, len(m.topIPs)-1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
