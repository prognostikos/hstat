package ui

import (
	"testing"
)

func TestCalculateLayout_MinimumSize(t *testing.T) {
	// Below minimum should return nil (too small)
	layout := CalculateLayout(59, 19)
	if layout != nil {
		t.Error("expected nil layout for terminal below minimum size")
	}

	// At minimum should work
	layout = CalculateLayout(60, 20)
	if layout == nil {
		t.Error("expected valid layout at minimum size 60x20")
	}
}

func TestCalculateLayout_SectionHeights(t *testing.T) {
	layout := CalculateLayout(120, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	// Header should have fixed height (3 lines content + 2 border)
	if layout.HeaderHeight < 5 {
		t.Errorf("expected HeaderHeight >= 5, got %d", layout.HeaderHeight)
	}

	// Status codes should have some height
	if layout.StatusCodesHeight < 3 {
		t.Errorf("expected StatusCodesHeight >= 3, got %d", layout.StatusCodesHeight)
	}

	// Data sections should get remaining space
	if layout.DataSectionHeight < 5 {
		t.Errorf("expected DataSectionHeight >= 5, got %d", layout.DataSectionHeight)
	}

	// Total should equal terminal height
	totalUsed := layout.HeaderHeight + layout.StatusCodesHeight + layout.DataSectionHeight
	if totalUsed != 40 {
		t.Errorf("expected total height 40, got %d", totalUsed)
	}
}

func TestCalculateLayout_WideTerminal(t *testing.T) {
	// Wide terminal: 3-column layout for data sections
	layout := CalculateLayout(180, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	if layout.DataColumns != 3 {
		t.Errorf("expected 3 columns for wide terminal (180), got %d", layout.DataColumns)
	}
}

func TestCalculateLayout_MediumTerminal(t *testing.T) {
	// Medium terminal: 2+1 layout (hosts/IPs side by side, paths below)
	layout := CalculateLayout(100, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	if layout.DataColumns != 2 {
		t.Errorf("expected 2 columns for medium terminal (100), got %d", layout.DataColumns)
	}
}

func TestCalculateLayout_NarrowTerminal(t *testing.T) {
	// Narrow terminal: stacked layout
	layout := CalculateLayout(70, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	if layout.DataColumns != 1 {
		t.Errorf("expected 1 column for narrow terminal (70), got %d", layout.DataColumns)
	}
}

func TestCalculateLayout_ColumnWidths(t *testing.T) {
	layout := CalculateLayout(180, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	// With 3 columns, each should get roughly 1/3 of width minus borders/gaps
	// Paths should get more width since paths are longer
	if layout.HostsWidth < 30 {
		t.Errorf("expected HostsWidth >= 30, got %d", layout.HostsWidth)
	}
	if layout.IPsWidth < 25 {
		t.Errorf("expected IPsWidth >= 25, got %d", layout.IPsWidth)
	}
	if layout.PathsWidth < 40 {
		t.Errorf("expected PathsWidth >= 40, got %d", layout.PathsWidth)
	}

	// Total width should not exceed terminal width
	totalWidth := layout.HostsWidth + layout.IPsWidth + layout.PathsWidth
	if totalWidth > 180 {
		t.Errorf("expected total width <= 180, got %d", totalWidth)
	}
}

func TestCalculateLayout_RowCounts(t *testing.T) {
	layout := CalculateLayout(120, 50)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	// Each data section should show multiple rows
	if layout.HostsRows < 3 {
		t.Errorf("expected HostsRows >= 3, got %d", layout.HostsRows)
	}
	if layout.IPsRows < 3 {
		t.Errorf("expected IPsRows >= 3, got %d", layout.IPsRows)
	}
	if layout.PathsRows < 3 {
		t.Errorf("expected PathsRows >= 3, got %d", layout.PathsRows)
	}
}

func TestCalculateLayout_StatusCodesColumns(t *testing.T) {
	// Wide terminal should fit all 5 status code columns
	layout := CalculateLayout(150, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	if layout.StatusCodeColumns != 5 {
		t.Errorf("expected 5 status code columns for wide terminal, got %d", layout.StatusCodeColumns)
	}

	// Narrow terminal might need fewer columns
	layout = CalculateLayout(80, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	// Should still have at least 3 columns
	if layout.StatusCodeColumns < 3 {
		t.Errorf("expected at least 3 status code columns, got %d", layout.StatusCodeColumns)
	}
}

func TestCalculateLayout_ActiveSectionPriority(t *testing.T) {
	// When space is tight and a section is active, it should get more rows
	layout := CalculateLayoutWithActiveSection(80, 25, SectionHosts)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	layoutIPs := CalculateLayoutWithActiveSection(80, 25, SectionIPs)
	if layoutIPs == nil {
		t.Fatal("expected valid layout")
	}

	// The active section should get at least as many or more rows
	if layout.HostsRows < layoutIPs.HostsRows {
		t.Errorf("expected HostsRows to be >= when hosts is active, hosts active: %d, ips active: %d",
			layout.HostsRows, layoutIPs.HostsRows)
	}
	if layoutIPs.IPsRows < layout.IPsRows {
		t.Errorf("expected IPsRows to be >= when ips is active, ips active: %d, hosts active: %d",
			layoutIPs.IPsRows, layout.IPsRows)
	}
}

func TestCalculateLayout_MediumTerminalPathsFullWidth(t *testing.T) {
	// In 2+1 mode, paths should span full width below hosts/IPs
	layout := CalculateLayout(100, 40)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	// Paths width should be close to full terminal width (minus borders)
	if layout.PathsWidth < 90 {
		t.Errorf("expected PathsWidth >= 90 in 2+1 mode, got %d", layout.PathsWidth)
	}
}

func TestCalculateLayout_VeryTallTerminal(t *testing.T) {
	// Very tall terminal should show many rows
	layout := CalculateLayout(120, 100)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	// Should show many items when there's lots of vertical space
	if layout.HostsRows < 20 {
		t.Errorf("expected many HostsRows for tall terminal, got %d", layout.HostsRows)
	}
}

func TestCalculateLayout_MinimalHeight(t *testing.T) {
	// At minimum height, should still show at least some rows
	layout := CalculateLayout(120, 20)
	if layout == nil {
		t.Fatal("expected valid layout")
	}

	// Each section should have at least 1 row
	if layout.HostsRows < 1 {
		t.Errorf("expected at least 1 HostsRow, got %d", layout.HostsRows)
	}
	if layout.IPsRows < 1 {
		t.Errorf("expected at least 1 IPsRow, got %d", layout.IPsRows)
	}
	if layout.PathsRows < 1 {
		t.Errorf("expected at least 1 PathsRow, got %d", layout.PathsRows)
	}
}
