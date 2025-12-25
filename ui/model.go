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

// Default number of items to show (will be dynamic based on layout)
const defaultTopN = 20

// Model is the bubbletea model
type Model struct {
	store       *store.Store
	startTime   time.Time
	refreshRate time.Duration

	// UI state
	width         int
	height        int
	section       Section
	hostCursor    int
	ipCursor      int
	filter        Filter
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

	// Additional stats
	rate4xx      float64
	rate5xx      float64
	uniqueHosts  int
	uniqueIPs    int
	uniquePaths  int
	currentRate  float64
	trend        store.Trend
	trend5m      store.Trend
	hostErrRates map[string]store.ErrorRates
	ipErrRates   map[string]store.ErrorRates
	pathErrRates map[string]store.ErrorRates
}

// NewModel creates a new Model
func NewModel(s *store.Store, refreshRate time.Duration) Model {
	return Model{
		store:       s,
		startTime:   time.Now(),
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

const currentRateWindow = 10 * time.Second
const trendWindow = 60 * time.Second
const trendWindow5m = 5 * time.Minute

// refreshData updates cached data from the store
func (m *Model) refreshData() {
	m.store.Prune()
	m.stats = m.store.GetStats()
	m.statusCounts = m.store.GetStatusCounts(m.filter.Host, m.filter.IP)

	// Use defaultTopN for now - will be dynamic based on layout in the future
	topN := defaultTopN
	m.topHosts = m.store.GetTopHosts(topN, m.filter.IP)
	m.topIPs = m.store.GetTopIPs(topN, m.filter.Host)

	// Get paths - always visible, filtered when host/IP is selected
	if m.filter.Host != "" || m.filter.IP != "" {
		m.topPaths = m.store.GetTopPaths(topN, m.filter.Host, m.filter.IP)
	} else {
		m.topPaths = m.store.GetAllPaths(topN)
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

	// Additional stats
	m.rate4xx, m.rate5xx = m.store.GetErrorRates()
	m.uniqueHosts, m.uniqueIPs, m.uniquePaths = m.store.GetUniqueCounts()
	m.currentRate = m.store.GetCurrentRate(currentRateWindow)

	// Update trends with hysteresis to prevent flickering
	m.trend = updateTrendWithHysteresis(m.trend, m.store, trendWindow)
	m.trend5m = updateTrendWithHysteresis(m.trend5m, m.store, trendWindow5m)

	// Error rates per host/IP/path
	m.hostErrRates = make(map[string]store.ErrorRates)
	for _, h := range m.topHosts {
		m.hostErrRates[h.Label] = m.store.GetErrorRatesForHost(h.Label)
	}
	m.ipErrRates = make(map[string]store.ErrorRates)
	for _, ip := range m.topIPs {
		m.ipErrRates[ip.Label] = m.store.GetErrorRatesForIP(ip.Label)
	}
	m.pathErrRates = make(map[string]store.ErrorRates)
	for _, p := range m.topPaths {
		m.pathErrRates[p.Label] = m.store.GetErrorRatesForPath(p.Label)
	}

	// Clamp cursors
	if m.hostCursor >= len(m.topHosts) {
		m.hostCursor = max(0, len(m.topHosts)-1)
	}
	if m.ipCursor >= len(m.topIPs) {
		m.ipCursor = max(0, len(m.topIPs)-1)
	}
}

// updateTrendWithHysteresis applies hysteresis to prevent trend flickering
// To enter a trend state requires 2% threshold, but to exit back to stable
// requires the diff to drop below 1%
func updateTrendWithHysteresis(current store.Trend, s *store.Store, period time.Duration) store.Trend {
	diff, newTrend := s.GetTrendWithDiff(period)

	// If new calculation shows a clear trend, always follow it
	if newTrend != store.TrendStable {
		return newTrend
	}

	// New calculation says stable - apply hysteresis
	// Only return to stable if diff is well within the threshold (< 1%)
	if current == store.TrendStable {
		return store.TrendStable
	}

	// Currently showing a trend, only clear it if diff is clearly below threshold
	absDiff := diff
	if absDiff < 0 {
		absDiff = -absDiff
	}
	if absDiff < 0.01 {
		return store.TrendStable
	}

	// Keep showing the current trend (sticky)
	return current
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
