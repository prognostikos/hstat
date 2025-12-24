package store

import (
	"sort"
	"sync"
	"time"

	"github.com/betternow/hstat/parser"
)

const maxEntries = 100000

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
	s.serviceTimes = append(s.serviceTimes, e.Service)
	s.connectTimes = append(s.connectTimes, e.Connect)

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
	}

	s.entries = s.entries[count:]
	s.serviceTimes = s.serviceTimes[count:]
	s.connectTimes = s.connectTimes[count:]
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
