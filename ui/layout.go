package ui

// Layout contains calculated dimensions for the UI
type Layout struct {
	// Terminal dimensions
	Width  int
	Height int

	// Section heights
	HeaderHeight      int
	StatusCodesHeight int
	DataSectionHeight int

	// Data section layout
	DataColumns int // 1=stacked, 2=2+1, 3=3-column

	// Column widths
	HostsWidth int
	IPsWidth   int
	PathsWidth int

	// Row counts per section
	HostsRows int
	IPsRows   int
	PathsRows int

	// Status codes layout
	StatusCodeColumns int
}

// Minimum terminal size
const (
	MinWidth  = 60
	MinHeight = 20
)

// Minimum column widths for data sections
const (
	MinHostsWidth = 30
	MinIPsWidth   = 25
	MinPathsWidth = 40
)

// CalculateLayout computes the layout based on terminal dimensions
func CalculateLayout(width, height int) *Layout {
	return CalculateLayoutWithActiveSection(width, height, SectionHosts)
}

// CalculateLayoutWithActiveSection computes layout with priority for active section
func CalculateLayoutWithActiveSection(width, height int, activeSection Section) *Layout {
	if width < MinWidth || height < MinHeight {
		return nil
	}

	layout := &Layout{
		Width:  width,
		Height: height,
	}

	// Calculate header height (title + 2 stats lines + borders)
	layout.HeaderHeight = 5

	// Calculate status codes height (header + group row + up to 3 detail rows + borders)
	layout.StatusCodesHeight = calculateStatusCodesHeight(width)

	// Remaining height for data sections
	layout.DataSectionHeight = height - layout.HeaderHeight - layout.StatusCodesHeight
	if layout.DataSectionHeight < 3 {
		layout.DataSectionHeight = 3
	}

	// Determine column layout based on width
	layout.DataColumns = calculateDataColumns(width)

	// Calculate column widths
	calculateColumnWidths(layout)

	// Calculate row counts
	calculateRowCounts(layout, activeSection)

	// Status code columns
	layout.StatusCodeColumns = calculateStatusCodeColumns(width)

	return layout
}

func calculateStatusCodesHeight(width int) int {
	// Header row + group percentages row + up to 3 detail rows + 2 border lines
	// For now, use a fixed height that works for most cases
	// This will be refined based on actual status code data
	return 7
}

func calculateDataColumns(width int) int {
	// Calculate how many columns fit based on minimum widths
	// 3 columns need: MinHostsWidth + MinIPsWidth + MinPathsWidth + borders/gaps
	threeColMinWidth := MinHostsWidth + MinIPsWidth + MinPathsWidth + 12 // borders and gaps

	// 2 columns need comfortable space for hosts + IPs side by side
	// MinHostsWidth(30) + MinIPsWidth(25) + borders(8) + some padding = ~80
	twoColMinWidth := 80

	if width >= threeColMinWidth+30 { // +30 for comfortable 3-column
		return 3
	} else if width >= twoColMinWidth {
		return 2
	}
	return 1
}

func calculateColumnWidths(layout *Layout) {
	usableWidth := layout.Width - 2 // outer border

	switch layout.DataColumns {
	case 3:
		// Three columns: hosts | IPs | paths
		// Paths gets more space since paths are longer
		gapWidth := 6 // gaps between columns
		contentWidth := usableWidth - gapWidth

		// Distribute: hosts 25%, IPs 25%, paths 50%
		layout.HostsWidth = contentWidth * 25 / 100
		layout.IPsWidth = contentWidth * 25 / 100
		layout.PathsWidth = contentWidth - layout.HostsWidth - layout.IPsWidth

		// Ensure minimums
		if layout.HostsWidth < MinHostsWidth {
			layout.HostsWidth = MinHostsWidth
		}
		if layout.IPsWidth < MinIPsWidth {
			layout.IPsWidth = MinIPsWidth
		}
		if layout.PathsWidth < MinPathsWidth {
			layout.PathsWidth = MinPathsWidth
		}

	case 2:
		// Two columns for hosts/IPs, paths below at full width
		gapWidth := 4
		topContentWidth := usableWidth - gapWidth

		layout.HostsWidth = topContentWidth / 2
		layout.IPsWidth = topContentWidth - layout.HostsWidth
		layout.PathsWidth = usableWidth - 2 // full width minus border

	case 1:
		// All stacked, full width
		layout.HostsWidth = usableWidth - 2
		layout.IPsWidth = usableWidth - 2
		layout.PathsWidth = usableWidth - 2
	}
}

func calculateRowCounts(layout *Layout, activeSection Section) {
	// Height available for data section content (minus borders)
	availableHeight := layout.DataSectionHeight - 2 // top/bottom borders

	switch layout.DataColumns {
	case 3:
		// All three side by side, equal height
		// Subtract 1 for header row in each section
		contentRows := availableHeight - 1
		if contentRows < 1 {
			contentRows = 1
		}

		// Distribute rows, with slight priority to active section
		baseRows := contentRows
		layout.HostsRows = baseRows
		layout.IPsRows = baseRows
		layout.PathsRows = baseRows

	case 2:
		// Hosts/IPs share top half, paths gets bottom half
		topHeight := (availableHeight * 55) / 100 // slightly more for top
		bottomHeight := availableHeight - topHeight

		topContentRows := topHeight - 1 // header row
		bottomContentRows := bottomHeight - 1

		if topContentRows < 1 {
			topContentRows = 1
		}
		if bottomContentRows < 1 {
			bottomContentRows = 1
		}

		// Active section gets more rows when space is tight
		if topContentRows < 5 {
			if activeSection == SectionHosts {
				layout.HostsRows = (topContentRows*3 + 2) / 4
				layout.IPsRows = topContentRows - layout.HostsRows + 1
			} else {
				layout.IPsRows = (topContentRows*3 + 2) / 4
				layout.HostsRows = topContentRows - layout.IPsRows + 1
			}
		} else {
			layout.HostsRows = topContentRows
			layout.IPsRows = topContentRows
		}
		layout.PathsRows = bottomContentRows

	case 1:
		// All stacked - divide equally with priority to active
		perSection := (availableHeight - 3) / 3 // 3 header rows
		if perSection < 1 {
			perSection = 1
		}

		// Give extra rows to active section when tight
		if perSection < 5 {
			extraRows := availableHeight - 3 - (perSection * 3)
			layout.HostsRows = perSection
			layout.IPsRows = perSection
			layout.PathsRows = perSection

			if activeSection == SectionHosts && extraRows > 0 {
				layout.HostsRows += extraRows
			} else if activeSection == SectionIPs && extraRows > 0 {
				layout.IPsRows += extraRows
			} else if extraRows > 0 {
				layout.PathsRows += extraRows
			}
		} else {
			layout.HostsRows = perSection
			layout.IPsRows = perSection
			layout.PathsRows = perSection
		}
	}
}

func calculateStatusCodeColumns(width int) int {
	// Each status column needs roughly 20 chars (category + individual codes)
	// Plus borders and padding
	colWidth := 20
	available := width - 4 // borders

	cols := available / colWidth
	if cols > 5 {
		cols = 5 // max 5 columns (1xx, 2xx, 3xx, 4xx, 5xx)
	}
	if cols < 1 {
		cols = 1
	}

	return cols
}
