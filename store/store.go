package store

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/betternow/hstat/parser"
)

const maxEntries = 100000

// Paths to exclude from display
var excludedPaths = []string{
	"/ahoy/events",
	"/ahoy/visits",
	"/robots.txt",
}

var excludedPathPrefixes = []string{
	"/system-status-",
	"/hirefire",
}

// isExcludedPath returns true if the path should be hidden from display
func isExcludedPath(path string) bool {
	for _, excluded := range excludedPaths {
		if path == excluded {
			return true
		}
	}
	for _, prefix := range excludedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Store holds time-windowed log data with pre-computed aggregates
type Store struct {
	mu      sync.RWMutex
	entries []parser.Entry
	window  time.Duration // 0 = keep all (up to maxEntries)

	// Aggregates
	TotalCount   int64
	StatusCounts map[int]int64
	HostCounts   map[string]int64
	IPCounts     map[string]int64

	// For percentiles
	serviceTimes []int
	connectTimes []int

	// For filtered views
	hostToIPs    map[string]map[string]int64 // host -> ip -> count
	ipToHosts    map[string]map[string]int64 // ip -> host -> count
	hostToStatus map[string]map[int]int64    // host -> status -> count
	ipToStatus   map[string]map[int]int64    // ip -> status -> count
	hostToPaths  map[string]map[string]int64 // host -> path -> count
	ipToPaths    map[string]map[string]int64 // ip -> path -> count
}

// New creates a new Store with the given window duration
func New(window time.Duration) *Store {
	return &Store{
		window:       window,
		StatusCounts: make(map[int]int64),
		HostCounts:   make(map[string]int64),
		IPCounts:     make(map[string]int64),
		hostToIPs:    make(map[string]map[string]int64),
		ipToHosts:    make(map[string]map[string]int64),
		hostToStatus: make(map[string]map[int]int64),
		ipToStatus:   make(map[string]map[int]int64),
		hostToPaths:  make(map[string]map[string]int64),
		ipToPaths:    make(map[string]map[string]int64),
	}
}

// Add adds an entry to the store
func (s *Store) Add(e *parser.Entry) {
	if e == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Normalize empty values
	host := e.Host
	ip := e.IP
	if host == "" {
		host = "(unknown)"
	}
	if ip == "" {
		ip = "(unknown)"
	}

	s.entries = append(s.entries, *e)
	s.TotalCount++
	s.StatusCounts[e.Status]++
	s.HostCounts[host]++
	s.IPCounts[ip]++
	// Skip 101 (WebSocket upgrade) for response time stats - they skew percentiles
	if e.Status != 101 {
		s.serviceTimes = append(s.serviceTimes, e.Service)
		s.connectTimes = append(s.connectTimes, e.Connect)
	}

	// Track relationships
	if s.hostToIPs[host] == nil {
		s.hostToIPs[host] = make(map[string]int64)
	}
	s.hostToIPs[host][ip]++

	if s.ipToHosts[ip] == nil {
		s.ipToHosts[ip] = make(map[string]int64)
	}
	s.ipToHosts[ip][host]++

	if s.hostToStatus[host] == nil {
		s.hostToStatus[host] = make(map[int]int64)
	}
	s.hostToStatus[host][e.Status]++

	if s.ipToStatus[ip] == nil {
		s.ipToStatus[ip] = make(map[int]int64)
	}
	s.ipToStatus[ip][e.Status]++

	// Track paths per host and IP
	path := e.Path
	if path == "" {
		path = "(unknown)"
	}
	if s.hostToPaths[host] == nil {
		s.hostToPaths[host] = make(map[string]int64)
	}
	s.hostToPaths[host][path]++

	if s.ipToPaths[ip] == nil {
		s.ipToPaths[ip] = make(map[string]int64)
	}
	s.ipToPaths[ip][path]++

	// Cap at maxEntries
	if len(s.entries) > maxEntries {
		s.pruneOldest(len(s.entries) - maxEntries)
	}
}

// Prune removes entries older than the window
func (s *Store) Prune() {
	if s.window == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.window)
	pruneCount := 0
	for i, e := range s.entries {
		if e.Timestamp.After(cutoff) {
			pruneCount = i
			break
		}
	}

	if pruneCount > 0 {
		s.pruneOldest(pruneCount)
	}
}

func (s *Store) pruneOldest(count int) {
	if count <= 0 || count > len(s.entries) {
		return
	}

	// Count non-101 entries being pruned (they have timing data)
	timingCount := 0

	// Decrement counts for pruned entries
	for i := 0; i < count; i++ {
		e := s.entries[i]
		host := e.Host
		ip := e.IP
		if host == "" {
			host = "(unknown)"
		}
		if ip == "" {
			ip = "(unknown)"
		}

		s.TotalCount--
		s.StatusCounts[e.Status]--
		s.HostCounts[host]--
		s.IPCounts[ip]--

		if s.hostToIPs[host] != nil {
			s.hostToIPs[host][ip]--
		}
		if s.ipToHosts[ip] != nil {
			s.ipToHosts[ip][host]--
		}
		if s.hostToStatus[host] != nil {
			s.hostToStatus[host][e.Status]--
		}
		if s.ipToStatus[ip] != nil {
			s.ipToStatus[ip][e.Status]--
		}

		path := e.Path
		if path == "" {
			path = "(unknown)"
		}
		if s.hostToPaths[host] != nil {
			s.hostToPaths[host][path]--
		}
		if s.ipToPaths[ip] != nil {
			s.ipToPaths[ip][path]--
		}

		if e.Status != 101 {
			timingCount++
		}
	}

	s.entries = s.entries[count:]
	s.serviceTimes = s.serviceTimes[timingCount:]
	s.connectTimes = s.connectTimes[timingCount:]
}

// Stats returns computed statistics
type Stats struct {
	TotalCount int64
	AvgService int
	P50Service int
	P95Service int
	P99Service int
	MaxService int
	AvgConnect int
	MaxConnect int
}

// GetStats returns current statistics
func (s *Store) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{TotalCount: s.TotalCount}

	if len(s.serviceTimes) == 0 {
		return stats
	}

	// Make a copy for sorting
	times := make([]int, len(s.serviceTimes))
	copy(times, s.serviceTimes)
	sort.Ints(times)

	// Avg
	sum := 0
	for _, t := range times {
		sum += t
	}
	stats.AvgService = sum / len(times)

	// Percentiles
	stats.P50Service = times[len(times)*50/100]
	stats.P95Service = times[len(times)*95/100]
	p99idx := len(times) * 99 / 100
	if p99idx >= len(times) {
		p99idx = len(times) - 1
	}
	stats.P99Service = times[p99idx]
	stats.MaxService = times[len(times)-1]

	// Connect times
	if len(s.connectTimes) > 0 {
		connSum := 0
		maxConn := 0
		for _, t := range s.connectTimes {
			connSum += t
			if t > maxConn {
				maxConn = t
			}
		}
		stats.AvgConnect = connSum / len(s.connectTimes)
		stats.MaxConnect = maxConn
	}

	return stats
}

// CountItem represents a count with label
type CountItem struct {
	Label string
	Count int64
}

// StatusCountItem represents a status code count
type StatusCountItem struct {
	Status int
	Count  int64
}

// GetStatusCounts returns status counts sorted by status code
func (s *Store) GetStatusCounts(filterHost, filterIP string) []StatusCountItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var counts map[int]int64

	if filterHost != "" {
		counts = s.hostToStatus[filterHost]
	} else if filterIP != "" {
		counts = s.ipToStatus[filterIP]
	} else {
		counts = s.StatusCounts
	}

	if counts == nil {
		return nil
	}

	items := make([]StatusCountItem, 0, len(counts))
	for status, count := range counts {
		if count > 0 {
			items = append(items, StatusCountItem{Status: status, Count: count})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Status < items[j].Status
	})

	return items
}

// GetTopHosts returns top N hosts by count
func (s *Store) GetTopHosts(n int, filterIP string) []CountItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var counts map[string]int64

	if filterIP != "" {
		counts = s.ipToHosts[filterIP]
	} else {
		counts = s.HostCounts
	}

	return s.topN(counts, n)
}

// GetTopIPs returns top N IPs by count
func (s *Store) GetTopIPs(n int, filterHost string) []CountItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var counts map[string]int64

	if filterHost != "" {
		counts = s.hostToIPs[filterHost]
	} else {
		counts = s.IPCounts
	}

	return s.topN(counts, n)
}

// GetTopPaths returns top N paths for a given host or IP
func (s *Store) GetTopPaths(n int, host, ip string) []CountItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var counts map[string]int64

	if host != "" {
		counts = s.hostToPaths[host]
	} else if ip != "" {
		counts = s.ipToPaths[ip]
	} else {
		return nil
	}

	// Filter out excluded paths
	filtered := make(map[string]int64)
	for path, count := range counts {
		if !isExcludedPath(path) {
			filtered[path] = count
		}
	}

	return s.topN(filtered, n)
}

func (s *Store) topN(counts map[string]int64, n int) []CountItem {
	if counts == nil {
		return nil
	}

	items := make([]CountItem, 0, len(counts))
	for label, count := range counts {
		if count > 0 {
			items = append(items, CountItem{Label: label, Count: count})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	if len(items) > n {
		items = items[:n]
	}

	return items
}

// GetOtherCount returns count of items not in top N
func (s *Store) GetOtherCount(counts map[string]int64, topN []CountItem) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	topSet := make(map[string]bool)
	for _, item := range topN {
		topSet[item.Label] = true
	}

	var other int64
	for label, count := range counts {
		if !topSet[label] && count > 0 {
			other += count
		}
	}
	return other
}

// StartTime returns when the first entry was recorded
func (s *Store) StartTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 {
		return time.Now()
	}
	return s.entries[0].Timestamp
}

// GetErrorRates returns the percentage of 4xx and 5xx responses
func (s *Store) GetErrorRates() (rate4xx, rate5xx float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalCount == 0 {
		return 0, 0
	}

	var count4xx, count5xx int64
	for status, count := range s.StatusCounts {
		if status >= 400 && status < 500 {
			count4xx += count
		} else if status >= 500 && status < 600 {
			count5xx += count
		}
	}

	rate4xx = float64(count4xx) * 100 / float64(s.TotalCount)
	rate5xx = float64(count5xx) * 100 / float64(s.TotalCount)
	return
}

// GetUniqueCounts returns the count of unique hosts, IPs, and paths
func (s *Store) GetUniqueCounts() (hosts, ips, paths int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, count := range s.HostCounts {
		if count > 0 {
			hosts++
		}
	}
	for _, count := range s.IPCounts {
		if count > 0 {
			ips++
		}
	}

	// Count unique paths across all hosts
	pathSet := make(map[string]bool)
	for _, pathCounts := range s.hostToPaths {
		for path, count := range pathCounts {
			if count > 0 {
				pathSet[path] = true
			}
		}
	}
	paths = len(pathSet)

	return
}

// GetCurrentRate returns the request rate over the given window
func (s *Store) GetCurrentRate(window time.Duration) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 {
		return 0
	}

	cutoff := time.Now().Add(-window)
	count := 0

	// Count entries within the window (iterate backwards for efficiency)
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].Timestamp.After(cutoff) {
			count++
		} else {
			break
		}
	}

	return float64(count) / window.Seconds()
}

// ErrorRates holds separate 4xx and 5xx error rates
type ErrorRates struct {
	Rate4xx float64
	Rate5xx float64
}

// GetErrorRatesForHost returns separate 4xx and 5xx rates for a specific host
func (s *Store) GetErrorRatesForHost(host string) ErrorRates {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.calculateErrorRates(s.hostToStatus[host])
}

// GetErrorRatesForIP returns separate 4xx and 5xx rates for a specific IP
func (s *Store) GetErrorRatesForIP(ip string) ErrorRates {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.calculateErrorRates(s.ipToStatus[ip])
}

func (s *Store) calculateErrorRates(statusCounts map[int]int64) ErrorRates {
	if statusCounts == nil {
		return ErrorRates{}
	}

	var total, count4xx, count5xx int64
	for status, count := range statusCounts {
		if count > 0 {
			total += count
			if status >= 400 && status < 500 {
				count4xx += count
			} else if status >= 500 {
				count5xx += count
			}
		}
	}

	if total == 0 {
		return ErrorRates{}
	}

	return ErrorRates{
		Rate4xx: float64(count4xx) * 100 / float64(total),
		Rate5xx: float64(count5xx) * 100 / float64(total),
	}
}

// Trend represents error rate trend direction
type Trend int

const (
	TrendStable Trend = iota
	TrendUp           // Error rate increasing (bad)
	TrendDown         // Error rate decreasing (good)
)

// GetTrend compares error rate in recent period vs previous period
// Returns the trend and the rate difference for hysteresis handling
func (s *Store) GetTrend(period time.Duration) Trend {
	_, trend := s.GetTrendWithDiff(period)
	return trend
}

// GetTrendWithDiff returns both the rate difference and computed trend
func (s *Store) GetTrendWithDiff(period time.Duration) (float64, Trend) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 {
		return 0, TrendStable
	}

	now := time.Now()
	recentCutoff := now.Add(-period)
	oldCutoff := now.Add(-2 * period)

	var recentTotal, recentErrors int64
	var oldTotal, oldErrors int64

	for _, e := range s.entries {
		isError := e.Status >= 400

		if e.Timestamp.After(recentCutoff) {
			recentTotal++
			if isError {
				recentErrors++
			}
		} else if e.Timestamp.After(oldCutoff) {
			oldTotal++
			if isError {
				oldErrors++
			}
		}
	}

	// Need sufficient data in both periods
	if recentTotal < 10 || oldTotal < 10 {
		return 0, TrendStable
	}

	recentRate := float64(recentErrors) / float64(recentTotal)
	oldRate := float64(oldErrors) / float64(oldTotal)

	diff := recentRate - oldRate

	// Use 2 percentage points as threshold for significance
	if diff > 0.02 {
		return diff, TrendUp
	} else if diff < -0.02 {
		return diff, TrendDown
	}

	return diff, TrendStable
}

// addEntryAtTime is a helper for testing - adds entry with specific timestamp
func (s *Store) addEntryAtTime(e *parser.Entry, t time.Time) {
	if e == nil {
		return
	}
	e.Timestamp = t
	s.Add(e)
}

// GetAllPaths returns top N paths across all hosts/IPs
func (s *Store) GetAllPaths(n int) []CountItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Aggregate all paths across all hosts, excluding hidden paths
	pathCounts := make(map[string]int64)
	for _, paths := range s.hostToPaths {
		for path, count := range paths {
			if count > 0 && !isExcludedPath(path) {
				pathCounts[path] += count
			}
		}
	}

	return s.topN(pathCounts, n)
}

// GetErrorRatesForPath returns separate 4xx and 5xx rates for a specific path
func (s *Store) GetErrorRatesForPath(path string) ErrorRates {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// We need to track path-to-status mapping
	// For now, we'll iterate through entries
	var total, count4xx, count5xx int64

	for _, e := range s.entries {
		p := e.Path
		if p == "" {
			p = "(unknown)"
		}
		if p == path {
			total++
			if e.Status >= 400 && e.Status < 500 {
				count4xx++
			} else if e.Status >= 500 {
				count5xx++
			}
		}
	}

	if total == 0 {
		return ErrorRates{}
	}

	return ErrorRates{
		Rate4xx: float64(count4xx) * 100 / float64(total),
		Rate5xx: float64(count5xx) * 100 / float64(total),
	}
}
