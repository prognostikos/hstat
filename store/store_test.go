package store

import (
	"testing"
	"time"

	"github.com/betternow/hstat/parser"
)

func TestNew(t *testing.T) {
	s := New(5 * time.Minute)
	if s == nil {
		t.Fatal("expected store, got nil")
	}
	if s.window != 5*time.Minute {
		t.Errorf("expected window 5m, got %v", s.window)
	}
	if s.TotalCount != 0 {
		t.Errorf("expected TotalCount 0, got %d", s.TotalCount)
	}
}

func TestAdd_NilEntry(t *testing.T) {
	s := New(0)
	s.Add(nil)
	if s.TotalCount != 0 {
		t.Errorf("expected TotalCount 0 after adding nil, got %d", s.TotalCount)
	}
}

func TestAdd_SingleEntry(t *testing.T) {
	s := New(0)
	entry := &parser.Entry{
		Timestamp: time.Now(),
		Status:    200,
		Service:   25,
		Connect:   1,
		Host:      "example.com",
		IP:        "1.2.3.4",
	}

	s.Add(entry)

	if s.TotalCount != 1 {
		t.Errorf("expected TotalCount 1, got %d", s.TotalCount)
	}
	if s.StatusCounts[200] != 1 {
		t.Errorf("expected status 200 count 1, got %d", s.StatusCounts[200])
	}
	if s.HostCounts["example.com"] != 1 {
		t.Errorf("expected host count 1, got %d", s.HostCounts["example.com"])
	}
	if s.IPCounts["1.2.3.4"] != 1 {
		t.Errorf("expected IP count 1, got %d", s.IPCounts["1.2.3.4"])
	}
}

func TestAdd_EmptyHostAndIP(t *testing.T) {
	s := New(0)
	entry := &parser.Entry{
		Timestamp: time.Now(),
		Status:    200,
		Host:      "",
		IP:        "",
	}

	s.Add(entry)

	if s.HostCounts["(unknown)"] != 1 {
		t.Errorf("expected unknown host count 1, got %d", s.HostCounts["(unknown)"])
	}
	if s.IPCounts["(unknown)"] != 1 {
		t.Errorf("expected unknown IP count 1, got %d", s.IPCounts["(unknown)"])
	}
}

func TestAdd_MultipleEntries(t *testing.T) {
	s := New(0)

	for i := 0; i < 100; i++ {
		status := 200
		if i%10 == 0 {
			status = 500
		}
		s.Add(&parser.Entry{
			Timestamp: time.Now(),
			Status:    status,
			Service:   10 + i,
			Host:      "example.com",
			IP:        "1.2.3.4",
		})
	}

	if s.TotalCount != 100 {
		t.Errorf("expected TotalCount 100, got %d", s.TotalCount)
	}
	if s.StatusCounts[200] != 90 {
		t.Errorf("expected status 200 count 90, got %d", s.StatusCounts[200])
	}
	if s.StatusCounts[500] != 10 {
		t.Errorf("expected status 500 count 10, got %d", s.StatusCounts[500])
	}
}

func TestGetStats_Empty(t *testing.T) {
	s := New(0)
	stats := s.GetStats()

	if stats.TotalCount != 0 {
		t.Errorf("expected TotalCount 0, got %d", stats.TotalCount)
	}
	if stats.AvgService != 0 {
		t.Errorf("expected AvgService 0, got %d", stats.AvgService)
	}
}

func TestGetStats_Percentiles(t *testing.T) {
	s := New(0)

	// Add 100 entries with service times 1-100
	for i := 1; i <= 100; i++ {
		s.Add(&parser.Entry{
			Timestamp: time.Now(),
			Status:    200,
			Service:   i,
			Connect:   1,
		})
	}

	stats := s.GetStats()

	if stats.TotalCount != 100 {
		t.Errorf("expected TotalCount 100, got %d", stats.TotalCount)
	}

	// Avg should be ~50
	if stats.AvgService < 49 || stats.AvgService > 51 {
		t.Errorf("expected AvgService ~50, got %d", stats.AvgService)
	}

	// P50 should be ~50
	if stats.P50Service < 49 || stats.P50Service > 51 {
		t.Errorf("expected P50Service ~50, got %d", stats.P50Service)
	}

	// P95 should be ~95
	if stats.P95Service < 94 || stats.P95Service > 96 {
		t.Errorf("expected P95Service ~95, got %d", stats.P95Service)
	}

	// P99 should be ~99
	if stats.P99Service < 98 || stats.P99Service > 100 {
		t.Errorf("expected P99Service ~99, got %d", stats.P99Service)
	}

	// Max should be 100
	if stats.MaxService != 100 {
		t.Errorf("expected MaxService 100, got %d", stats.MaxService)
	}
}

func TestGetStatusCounts(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 404, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 500, Host: "b.com", IP: "2.2.2.2"})

	// Unfiltered
	counts := s.GetStatusCounts("", "")
	if len(counts) != 3 {
		t.Errorf("expected 3 status codes, got %d", len(counts))
	}

	// Should be sorted by status code
	if counts[0].Status != 200 || counts[0].Count != 2 {
		t.Errorf("expected status 200 count 2 first, got %d count %d", counts[0].Status, counts[0].Count)
	}

	// Filtered by host
	counts = s.GetStatusCounts("a.com", "")
	if len(counts) != 2 {
		t.Errorf("expected 2 status codes for host a.com, got %d", len(counts))
	}

	// Filtered by IP
	counts = s.GetStatusCounts("", "2.2.2.2")
	if len(counts) != 1 {
		t.Errorf("expected 1 status code for IP 2.2.2.2, got %d", len(counts))
	}
	if counts[0].Status != 500 {
		t.Errorf("expected status 500, got %d", counts[0].Status)
	}
}

func TestGetTopHosts(t *testing.T) {
	s := New(0)

	// Add entries with different host frequencies
	for i := 0; i < 10; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "top.com", IP: "1.1.1.1"})
	}
	for i := 0; i < 5; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "mid.com", IP: "1.1.1.1"})
	}
	for i := 0; i < 2; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "low.com", IP: "1.1.1.1"})
	}

	// Get top 2
	hosts := s.GetTopHosts(2, "")
	if len(hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Label != "top.com" || hosts[0].Count != 10 {
		t.Errorf("expected top.com with 10, got %s with %d", hosts[0].Label, hosts[0].Count)
	}
	if hosts[1].Label != "mid.com" || hosts[1].Count != 5 {
		t.Errorf("expected mid.com with 5, got %s with %d", hosts[1].Label, hosts[1].Count)
	}
}

func TestGetTopIPs(t *testing.T) {
	s := New(0)

	for i := 0; i < 10; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "example.com", IP: "1.1.1.1"})
	}
	for i := 0; i < 3; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "example.com", IP: "2.2.2.2"})
	}

	ips := s.GetTopIPs(10, "")
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(ips))
	}
	if ips[0].Label != "1.1.1.1" || ips[0].Count != 10 {
		t.Errorf("expected 1.1.1.1 with 10, got %s with %d", ips[0].Label, ips[0].Count)
	}
}

func TestGetTopHosts_FilteredByIP(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "b.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "c.com", IP: "2.2.2.2"})

	// Filter by IP 1.1.1.1 - should only see a.com and b.com
	hosts := s.GetTopHosts(10, "1.1.1.1")
	if len(hosts) != 2 {
		t.Errorf("expected 2 hosts for IP 1.1.1.1, got %d", len(hosts))
	}

	// a.com should be first (2 requests)
	if hosts[0].Label != "a.com" || hosts[0].Count != 2 {
		t.Errorf("expected a.com with 2, got %s with %d", hosts[0].Label, hosts[0].Count)
	}
}

func TestGetTopIPs_FilteredByHost(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "2.2.2.2"})
	s.Add(&parser.Entry{Status: 200, Host: "b.com", IP: "3.3.3.3"})

	// Filter by host a.com
	ips := s.GetTopIPs(10, "a.com")
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs for host a.com, got %d", len(ips))
	}

	// 1.1.1.1 should be first (2 requests)
	if ips[0].Label != "1.1.1.1" || ips[0].Count != 2 {
		t.Errorf("expected 1.1.1.1 with 2, got %s with %d", ips[0].Label, ips[0].Count)
	}
}

func TestGetOtherCount(t *testing.T) {
	s := New(0)

	for i := 0; i < 10; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "a.com"})
	}
	for i := 0; i < 5; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "b.com"})
	}
	for i := 0; i < 3; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "c.com"})
	}
	for i := 0; i < 2; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "d.com"})
	}

	// Get top 2 hosts
	topHosts := s.GetTopHosts(2, "")

	// Other should be c.com (3) + d.com (2) = 5
	other := s.GetOtherCount(s.HostCounts, topHosts)
	if other != 5 {
		t.Errorf("expected other count 5, got %d", other)
	}
}

func TestPrune_NoWindow(t *testing.T) {
	s := New(0) // No window = keep all

	for i := 0; i < 10; i++ {
		s.Add(&parser.Entry{Status: 200})
	}

	s.Prune()

	if s.TotalCount != 10 {
		t.Errorf("expected TotalCount 10 after prune with no window, got %d", s.TotalCount)
	}
}

func TestPrune_WithWindow(t *testing.T) {
	s := New(100 * time.Millisecond)

	// Add old entry
	oldEntry := &parser.Entry{
		Timestamp: time.Now().Add(-200 * time.Millisecond),
		Status:    200,
		Host:      "old.com",
		IP:        "1.1.1.1",
	}
	s.mu.Lock()
	s.entries = append(s.entries, *oldEntry)
	s.TotalCount++
	s.StatusCounts[200]++
	s.HostCounts["old.com"]++
	s.IPCounts["1.1.1.1"]++
	s.serviceTimes = append(s.serviceTimes, 0)
	s.connectTimes = append(s.connectTimes, 0)
	s.mu.Unlock()

	// Add new entry
	s.Add(&parser.Entry{
		Timestamp: time.Now(),
		Status:    200,
		Host:      "new.com",
		IP:        "2.2.2.2",
	})

	if s.TotalCount != 2 {
		t.Errorf("expected TotalCount 2 before prune, got %d", s.TotalCount)
	}

	s.Prune()

	if s.TotalCount != 1 {
		t.Errorf("expected TotalCount 1 after prune, got %d", s.TotalCount)
	}
	if s.HostCounts["old.com"] != 0 {
		t.Errorf("expected old.com count 0, got %d", s.HostCounts["old.com"])
	}
	if s.HostCounts["new.com"] != 1 {
		t.Errorf("expected new.com count 1, got %d", s.HostCounts["new.com"])
	}
}

func TestRelationships(t *testing.T) {
	s := New(0)

	// Multiple IPs hitting same host
	s.Add(&parser.Entry{Status: 200, Host: "api.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", IP: "2.2.2.2"})

	// Same IP hitting multiple hosts
	s.Add(&parser.Entry{Status: 200, Host: "web.com", IP: "1.1.1.1"})

	// Check hostToIPs
	if s.hostToIPs["api.com"]["1.1.1.1"] != 2 {
		t.Errorf("expected api.com->1.1.1.1 count 2, got %d", s.hostToIPs["api.com"]["1.1.1.1"])
	}
	if s.hostToIPs["api.com"]["2.2.2.2"] != 1 {
		t.Errorf("expected api.com->2.2.2.2 count 1, got %d", s.hostToIPs["api.com"]["2.2.2.2"])
	}

	// Check ipToHosts
	if s.ipToHosts["1.1.1.1"]["api.com"] != 2 {
		t.Errorf("expected 1.1.1.1->api.com count 2, got %d", s.ipToHosts["1.1.1.1"]["api.com"])
	}
	if s.ipToHosts["1.1.1.1"]["web.com"] != 1 {
		t.Errorf("expected 1.1.1.1->web.com count 1, got %d", s.ipToHosts["1.1.1.1"]["web.com"])
	}
}

func TestStatus101_ExcludedFromTimingStats(t *testing.T) {
	s := New(0)

	// Add normal entries with service times 10, 20, 30
	s.Add(&parser.Entry{Status: 200, Service: 10, Connect: 1})
	s.Add(&parser.Entry{Status: 200, Service: 20, Connect: 2})
	s.Add(&parser.Entry{Status: 200, Service: 30, Connect: 3})

	// Add 101 (WebSocket) with very high service time that would skew stats
	s.Add(&parser.Entry{Status: 101, Service: 100000, Connect: 50000})

	stats := s.GetStats()

	// TotalCount should include the 101
	if stats.TotalCount != 4 {
		t.Errorf("expected TotalCount 4, got %d", stats.TotalCount)
	}

	// But timing stats should only reflect the 200s
	if stats.AvgService != 20 {
		t.Errorf("expected AvgService 20 (excluding 101), got %d", stats.AvgService)
	}
	if stats.MaxService != 30 {
		t.Errorf("expected MaxService 30 (excluding 101), got %d", stats.MaxService)
	}
	if stats.AvgConnect != 2 {
		t.Errorf("expected AvgConnect 2 (excluding 101), got %d", stats.AvgConnect)
	}
	if stats.MaxConnect != 3 {
		t.Errorf("expected MaxConnect 3 (excluding 101), got %d", stats.MaxConnect)
	}
}

func TestStatus101_StillCountedInStatusCounts(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 101, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 101, Host: "a.com", IP: "1.1.1.1"})

	if s.StatusCounts[101] != 2 {
		t.Errorf("expected status 101 count 2, got %d", s.StatusCounts[101])
	}
	if s.StatusCounts[200] != 1 {
		t.Errorf("expected status 200 count 1, got %d", s.StatusCounts[200])
	}
	if s.TotalCount != 3 {
		t.Errorf("expected TotalCount 3, got %d", s.TotalCount)
	}
}

func TestPrune_WithStatus101(t *testing.T) {
	s := New(100 * time.Millisecond)

	// Manually add old entries - mix of 101 and 200
	now := time.Now()
	oldTime := now.Add(-200 * time.Millisecond)

	s.mu.Lock()
	// Old 200 entry (has timing data)
	s.entries = append(s.entries, parser.Entry{Timestamp: oldTime, Status: 200, Host: "old.com", IP: "1.1.1.1"})
	s.TotalCount++
	s.StatusCounts[200]++
	s.HostCounts["old.com"]++
	s.IPCounts["1.1.1.1"]++
	s.serviceTimes = append(s.serviceTimes, 10)
	s.connectTimes = append(s.connectTimes, 1)

	// Old 101 entry (no timing data)
	s.entries = append(s.entries, parser.Entry{Timestamp: oldTime, Status: 101, Host: "old.com", IP: "1.1.1.1"})
	s.TotalCount++
	s.StatusCounts[101]++
	s.HostCounts["old.com"]++
	s.IPCounts["1.1.1.1"]++
	// No serviceTimes/connectTimes for 101
	s.mu.Unlock()

	// Add new entries
	s.Add(&parser.Entry{Timestamp: now, Status: 200, Service: 20, Connect: 2, Host: "new.com", IP: "2.2.2.2"})

	if s.TotalCount != 3 {
		t.Errorf("expected TotalCount 3 before prune, got %d", s.TotalCount)
	}
	if len(s.serviceTimes) != 2 {
		t.Errorf("expected 2 service times before prune, got %d", len(s.serviceTimes))
	}

	s.Prune()

	// Should have pruned both old entries
	if s.TotalCount != 1 {
		t.Errorf("expected TotalCount 1 after prune, got %d", s.TotalCount)
	}
	// Should have pruned only 1 timing entry (the 200, not the 101)
	if len(s.serviceTimes) != 1 {
		t.Errorf("expected 1 service time after prune, got %d", len(s.serviceTimes))
	}
	if s.serviceTimes[0] != 20 {
		t.Errorf("expected remaining service time to be 20, got %d", s.serviceTimes[0])
	}
}

func TestPathTracking(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/orders", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "web.com", Path: "/home", IP: "1.1.1.1"})

	// Get paths for api.com
	paths := s.GetTopPaths(10, "api.com", "")
	if len(paths) != 2 {
		t.Errorf("expected 2 paths for api.com, got %d", len(paths))
	}

	// /users should be first (2 requests)
	if paths[0].Label != "/users" || paths[0].Count != 2 {
		t.Errorf("expected /users with 2, got %s with %d", paths[0].Label, paths[0].Count)
	}

	// /orders should be second (1 request)
	if paths[1].Label != "/orders" || paths[1].Count != 1 {
		t.Errorf("expected /orders with 1, got %s with %d", paths[1].Label, paths[1].Count)
	}
}

func TestPathTracking_NoHost(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})

	// Get paths for non-existent host
	paths := s.GetTopPaths(10, "other.com", "")
	if len(paths) != 0 {
		t.Errorf("expected 0 paths for other.com, got %d", len(paths))
	}
}

func TestPathTracking_EmptyPath(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})

	paths := s.GetTopPaths(10, "api.com", "")
	// Empty paths should be normalized to (unknown)
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}
}

func TestPathTracking_Prune(t *testing.T) {
	s := New(100 * time.Millisecond)

	now := time.Now()
	oldTime := now.Add(-200 * time.Millisecond)

	// Manually add old entry with path
	s.mu.Lock()
	s.entries = append(s.entries, parser.Entry{Timestamp: oldTime, Status: 200, Host: "api.com", Path: "/old", IP: "1.1.1.1"})
	s.TotalCount++
	s.StatusCounts[200]++
	s.HostCounts["api.com"]++
	s.IPCounts["1.1.1.1"]++
	s.serviceTimes = append(s.serviceTimes, 0)
	s.connectTimes = append(s.connectTimes, 0)
	if s.hostToPaths["api.com"] == nil {
		s.hostToPaths["api.com"] = make(map[string]int64)
	}
	s.hostToPaths["api.com"]["/old"]++
	s.mu.Unlock()

	// Add new entry
	s.Add(&parser.Entry{Timestamp: now, Status: 200, Host: "api.com", Path: "/new", IP: "1.1.1.1"})

	// Before prune
	paths := s.GetTopPaths(10, "api.com", "")
	if len(paths) != 2 {
		t.Errorf("expected 2 paths before prune, got %d", len(paths))
	}

	s.Prune()

	// After prune - only /new should remain
	paths = s.GetTopPaths(10, "api.com", "")
	if len(paths) != 1 {
		t.Errorf("expected 1 path after prune, got %d", len(paths))
	}
	if paths[0].Label != "/new" {
		t.Errorf("expected /new to remain, got %s", paths[0].Label)
	}
}

func TestPathTracking_ByIP(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/orders", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "api.com", Path: "/admin", IP: "2.2.2.2"})

	// Get paths for IP 1.1.1.1
	paths := s.GetTopPaths(10, "", "1.1.1.1")
	if len(paths) != 2 {
		t.Errorf("expected 2 paths for IP 1.1.1.1, got %d", len(paths))
	}

	// /users should be first (2 requests)
	if paths[0].Label != "/users" || paths[0].Count != 2 {
		t.Errorf("expected /users with 2, got %s with %d", paths[0].Label, paths[0].Count)
	}

	// /orders should be second (1 request)
	if paths[1].Label != "/orders" || paths[1].Count != 1 {
		t.Errorf("expected /orders with 1, got %s with %d", paths[1].Label, paths[1].Count)
	}

	// Get paths for IP 2.2.2.2
	paths = s.GetTopPaths(10, "", "2.2.2.2")
	if len(paths) != 1 {
		t.Errorf("expected 1 path for IP 2.2.2.2, got %d", len(paths))
	}
	if paths[0].Label != "/admin" {
		t.Errorf("expected /admin, got %s", paths[0].Label)
	}
}

func TestPathTracking_ByIP_Prune(t *testing.T) {
	s := New(100 * time.Millisecond)

	now := time.Now()
	oldTime := now.Add(-200 * time.Millisecond)

	// Manually add old entry with path
	s.mu.Lock()
	s.entries = append(s.entries, parser.Entry{Timestamp: oldTime, Status: 200, Host: "api.com", Path: "/old", IP: "1.1.1.1"})
	s.TotalCount++
	s.StatusCounts[200]++
	s.HostCounts["api.com"]++
	s.IPCounts["1.1.1.1"]++
	s.serviceTimes = append(s.serviceTimes, 0)
	s.connectTimes = append(s.connectTimes, 0)
	if s.hostToPaths["api.com"] == nil {
		s.hostToPaths["api.com"] = make(map[string]int64)
	}
	s.hostToPaths["api.com"]["/old"]++
	if s.ipToPaths["1.1.1.1"] == nil {
		s.ipToPaths["1.1.1.1"] = make(map[string]int64)
	}
	s.ipToPaths["1.1.1.1"]["/old"]++
	s.mu.Unlock()

	// Add new entry
	s.Add(&parser.Entry{Timestamp: now, Status: 200, Host: "api.com", Path: "/new", IP: "1.1.1.1"})

	// Before prune
	paths := s.GetTopPaths(10, "", "1.1.1.1")
	if len(paths) != 2 {
		t.Errorf("expected 2 paths before prune, got %d", len(paths))
	}

	s.Prune()

	// After prune - only /new should remain
	paths = s.GetTopPaths(10, "", "1.1.1.1")
	if len(paths) != 1 {
		t.Errorf("expected 1 path after prune, got %d", len(paths))
	}
	if paths[0].Label != "/new" {
		t.Errorf("expected /new to remain, got %s", paths[0].Label)
	}
}

func TestGetErrorRates(t *testing.T) {
	s := New(0)

	// Add mix of status codes
	for i := 0; i < 80; i++ {
		s.Add(&parser.Entry{Status: 200})
	}
	for i := 0; i < 10; i++ {
		s.Add(&parser.Entry{Status: 404})
	}
	for i := 0; i < 5; i++ {
		s.Add(&parser.Entry{Status: 500})
	}
	for i := 0; i < 5; i++ {
		s.Add(&parser.Entry{Status: 503})
	}

	rate4xx, rate5xx := s.GetErrorRates()

	// 10 out of 100 = 10% 4xx
	if rate4xx < 9.9 || rate4xx > 10.1 {
		t.Errorf("expected 4xx rate ~10%%, got %.1f%%", rate4xx)
	}

	// 10 out of 100 = 10% 5xx
	if rate5xx < 9.9 || rate5xx > 10.1 {
		t.Errorf("expected 5xx rate ~10%%, got %.1f%%", rate5xx)
	}
}

func TestGetErrorRates_Empty(t *testing.T) {
	s := New(0)
	rate4xx, rate5xx := s.GetErrorRates()

	if rate4xx != 0 || rate5xx != 0 {
		t.Errorf("expected 0%% error rates for empty store, got 4xx=%.1f%%, 5xx=%.1f%%", rate4xx, rate5xx)
	}
}

func TestGetUniqueCounts(t *testing.T) {
	s := New(0)

	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1", Path: "/users"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1", Path: "/users"})
	s.Add(&parser.Entry{Status: 200, Host: "b.com", IP: "2.2.2.2", Path: "/orders"})
	s.Add(&parser.Entry{Status: 200, Host: "c.com", IP: "1.1.1.1", Path: "/users"})

	hosts, ips, paths := s.GetUniqueCounts()

	if hosts != 3 {
		t.Errorf("expected 3 unique hosts, got %d", hosts)
	}
	if ips != 2 {
		t.Errorf("expected 2 unique IPs, got %d", ips)
	}
	if paths != 2 {
		t.Errorf("expected 2 unique paths, got %d", paths)
	}
}

func TestGetCurrentRate(t *testing.T) {
	s := New(0)

	now := time.Now()

	// Add 10 entries from 30 seconds ago first (chronological order)
	for i := 0; i < 10; i++ {
		s.addEntryAtTime(&parser.Entry{Status: 200}, now.Add(-30*time.Second))
	}

	// Add 10 entries in the last 5 seconds
	for i := 0; i < 10; i++ {
		s.addEntryAtTime(&parser.Entry{Status: 200}, now.Add(-time.Duration(i)*500*time.Millisecond))
	}

	rate := s.GetCurrentRate(10 * time.Second)

	// 10 entries in 10 seconds = 1.0 req/s
	if rate < 0.9 || rate > 1.1 {
		t.Errorf("expected rate ~1.0 req/s, got %.2f", rate)
	}
}

func TestGetErrorRatesForHost(t *testing.T) {
	s := New(0)

	// Host a.com: 8 success, 1 4xx, 1 5xx
	for i := 0; i < 8; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	}
	s.Add(&parser.Entry{Status: 404, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 500, Host: "a.com", IP: "1.1.1.1"})

	// Host b.com: all success
	for i := 0; i < 10; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "b.com", IP: "2.2.2.2"})
	}

	ratesA := s.GetErrorRatesForHost("a.com")
	ratesB := s.GetErrorRatesForHost("b.com")

	if ratesA.Rate4xx < 9.9 || ratesA.Rate4xx > 10.1 {
		t.Errorf("expected a.com 4xx rate ~10%%, got %.1f%%", ratesA.Rate4xx)
	}
	if ratesA.Rate5xx < 9.9 || ratesA.Rate5xx > 10.1 {
		t.Errorf("expected a.com 5xx rate ~10%%, got %.1f%%", ratesA.Rate5xx)
	}
	if ratesB.Rate4xx != 0 || ratesB.Rate5xx != 0 {
		t.Errorf("expected b.com error rates 0%%, got 4xx=%.1f%% 5xx=%.1f%%", ratesB.Rate4xx, ratesB.Rate5xx)
	}
}

func TestGetErrorRatesForIP(t *testing.T) {
	s := New(0)

	// IP 1.1.1.1: 8 success, 1 4xx, 1 5xx
	for i := 0; i < 8; i++ {
		s.Add(&parser.Entry{Status: 200, Host: "a.com", IP: "1.1.1.1"})
	}
	s.Add(&parser.Entry{Status: 404, Host: "a.com", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 503, Host: "a.com", IP: "1.1.1.1"})

	rates := s.GetErrorRatesForIP("1.1.1.1")

	if rates.Rate4xx < 9.9 || rates.Rate4xx > 10.1 {
		t.Errorf("expected 4xx rate ~10%%, got %.1f%%", rates.Rate4xx)
	}
	if rates.Rate5xx < 9.9 || rates.Rate5xx > 10.1 {
		t.Errorf("expected 5xx rate ~10%%, got %.1f%%", rates.Rate5xx)
	}
}

func TestGetTrend(t *testing.T) {
	s := New(0)

	now := time.Now()

	// Old period (30-60s ago): 10 requests, 1 error = 10%
	for i := 0; i < 9; i++ {
		s.addEntryAtTime(&parser.Entry{Status: 200}, now.Add(-45*time.Second))
	}
	s.addEntryAtTime(&parser.Entry{Status: 500}, now.Add(-45*time.Second))

	// Recent period (0-30s ago): 10 requests, 3 errors = 30%
	for i := 0; i < 7; i++ {
		s.addEntryAtTime(&parser.Entry{Status: 200}, now.Add(-15*time.Second))
	}
	for i := 0; i < 3; i++ {
		s.addEntryAtTime(&parser.Entry{Status: 500}, now.Add(-15*time.Second))
	}

	trend := s.GetTrend(30 * time.Second)

	// Error rate increased from 10% to 30%, trend should be positive (worsening)
	if trend != TrendUp {
		t.Errorf("expected TrendUp (error rate increased), got %v", trend)
	}
}

func BenchmarkAdd(b *testing.B) {
	s := New(5 * time.Minute)
	entry := &parser.Entry{
		Timestamp: time.Now(),
		Status:    200,
		Service:   25,
		Connect:   1,
		Host:      "example.com",
		IP:        "1.2.3.4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(entry)
	}
}

func BenchmarkGetStats(b *testing.B) {
	s := New(0)
	for i := 0; i < 10000; i++ {
		s.Add(&parser.Entry{
			Status:  200,
			Service: i % 1000,
			Connect: i % 100,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.GetStats()
	}
}

func TestGetAllPaths(t *testing.T) {
	s := New(0)

	// Add entries with different paths
	s.Add(&parser.Entry{Status: 200, Host: "a.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "b.com", Path: "/orders", IP: "2.2.2.2"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", Path: "/admin", IP: "1.1.1.1"})

	paths := s.GetAllPaths(10)

	if len(paths) != 3 {
		t.Errorf("expected 3 unique paths, got %d", len(paths))
	}

	// Should be sorted by count descending
	if paths[0].Label != "/users" {
		t.Errorf("expected first path to be /users, got %s", paths[0].Label)
	}
	if paths[0].Count != 2 {
		t.Errorf("expected /users count 2, got %d", paths[0].Count)
	}
}

func TestGetErrorRatesForPath(t *testing.T) {
	s := New(0)

	// Add entries with different statuses for paths
	s.Add(&parser.Entry{Status: 200, Host: "a.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 200, Host: "a.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 404, Host: "a.com", Path: "/users", IP: "1.1.1.1"})
	s.Add(&parser.Entry{Status: 500, Host: "a.com", Path: "/users", IP: "1.1.1.1"})

	rates := s.GetErrorRatesForPath("/users")

	// 1 out of 4 is 404, 1 out of 4 is 500
	expectedRate4xx := 25.0
	expectedRate5xx := 25.0

	if rates.Rate4xx != expectedRate4xx {
		t.Errorf("expected 4xx rate %.1f, got %.1f", expectedRate4xx, rates.Rate4xx)
	}
	if rates.Rate5xx != expectedRate5xx {
		t.Errorf("expected 5xx rate %.1f, got %.1f", expectedRate5xx, rates.Rate5xx)
	}
}

func TestGetAllPaths_Empty(t *testing.T) {
	s := New(0)
	paths := s.GetAllPaths(10)

	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestGetErrorRatesForPath_NotFound(t *testing.T) {
	s := New(0)
	rates := s.GetErrorRatesForPath("/nonexistent")

	if rates.Rate4xx != 0 || rates.Rate5xx != 0 {
		t.Error("expected zero rates for nonexistent path")
	}
}

func TestExcludedPaths(t *testing.T) {
	s := New(0)

	// Add some normal paths and excluded paths
	s.Add(&parser.Entry{Host: "a.com", Path: "/api/users", Status: 200, IP: "1.1.1.1"})
	s.Add(&parser.Entry{Host: "a.com", Path: "/ahoy/events", Status: 200, IP: "1.1.1.1"})
	s.Add(&parser.Entry{Host: "a.com", Path: "/ahoy/visits", Status: 200, IP: "1.1.1.1"})
	s.Add(&parser.Entry{Host: "a.com", Path: "/robots.txt", Status: 200, IP: "1.1.1.1"})
	s.Add(&parser.Entry{Host: "a.com", Path: "/system-status-abc", Status: 200, IP: "1.1.1.1"})
	s.Add(&parser.Entry{Host: "a.com", Path: "/hirefire/test", Status: 200, IP: "1.1.1.1"})
	s.Add(&parser.Entry{Host: "a.com", Path: "/api/orders", Status: 200, IP: "1.1.1.1"})

	// GetTopPaths should only return non-excluded paths
	paths := s.GetTopPaths(10, "a.com", "")
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}

	for _, p := range paths {
		if p.Label == "/ahoy/events" || p.Label == "/ahoy/visits" ||
			p.Label == "/robots.txt" || p.Label == "/system-status-abc" ||
			p.Label == "/hirefire/test" {
			t.Errorf("excluded path %s should not appear in results", p.Label)
		}
	}

	// GetAllPaths should also filter
	allPaths := s.GetAllPaths(10)
	if len(allPaths) != 2 {
		t.Errorf("expected 2 paths from GetAllPaths, got %d", len(allPaths))
	}
}
