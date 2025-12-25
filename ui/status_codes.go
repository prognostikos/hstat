package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/betternow/hstat/store"
)

// CodeData represents a single status code's data
type CodeData struct {
	Code       int
	Count      int64
	Percentage float64
}

// CategoryData represents a status code category (1xx, 2xx, etc.)
type CategoryData struct {
	Total      int64
	Percentage float64
	Codes      []CodeData
}

// StatusCodesData holds all status code data for rendering
type StatusCodesData struct {
	Categories map[int]CategoryData // key is category number (1, 2, 3, 4, 5)
}

// StatusCodesDataFromStore converts store status counts to StatusCodesData
func StatusCodesDataFromStore(counts []store.StatusCountItem) StatusCodesData {
	data := StatusCodesData{
		Categories: make(map[int]CategoryData),
	}

	// Calculate total for percentages
	var total int64
	for _, sc := range counts {
		total += sc.Count
	}

	// Group by category
	categoryTotals := make(map[int]int64)
	categoryCodes := make(map[int][]CodeData)

	for _, sc := range counts {
		cat := sc.Status / 100
		categoryTotals[cat] += sc.Count

		pct := float64(0)
		if total > 0 {
			pct = float64(sc.Count) * 100 / float64(total)
		}

		categoryCodes[cat] = append(categoryCodes[cat], CodeData{
			Code:       sc.Status,
			Count:      sc.Count,
			Percentage: pct,
		})
	}

	// Build category data
	for cat := 1; cat <= 5; cat++ {
		catTotal := categoryTotals[cat]
		catPct := float64(0)
		if total > 0 {
			catPct = float64(catTotal) * 100 / float64(total)
		}

		codes := categoryCodes[cat]
		// Sort codes by count descending
		sort.Slice(codes, func(i, j int) bool {
			return codes[i].Count > codes[j].Count
		})

		data.Categories[cat] = CategoryData{
			Total:      catTotal,
			Percentage: catPct,
			Codes:      codes,
		}
	}

	return data
}

// RenderStatusCodesColumnar renders status codes in a columnar layout
func RenderStatusCodesColumnar(data StatusCodesData, width int, maxDetailRows int) string {
	numColumns := calculateStatusCodeColumns(width)
	colWidth := (width - 4) / numColumns // account for borders/padding

	// Get categories that have data (or all 5 for consistency)
	categories := []int{1, 2, 3, 4, 5}

	// Calculate how many rows we need for column layout
	categoriesPerRow := numColumns
	numCatRows := (len(categories) + categoriesPerRow - 1) / categoriesPerRow

	var lines []string

	// Render in rows of columns
	for rowIdx := 0; rowIdx < numCatRows; rowIdx++ {
		startCat := rowIdx * categoriesPerRow
		endCat := startCat + categoriesPerRow
		if endCat > len(categories) {
			endCat = len(categories)
		}

		rowCategories := categories[startCat:endCat]

		// Header row: "1xx (%)  2xx (%)  ..."
		headerParts := make([]string, len(rowCategories))
		for i, cat := range rowCategories {
			catData := data.Categories[cat]
			pctStr := "-"
			if catData.Percentage > 0 {
				pctStr = fmt.Sprintf("%.1f%%", catData.Percentage)
			}
			header := fmt.Sprintf("%dxx (%s)", cat, pctStr)
			headerParts[i] = padToWidth(header, colWidth)
		}
		lines = append(lines, "  "+strings.Join(headerParts, "  "))

		// Find max detail rows for this row of categories
		maxCodes := 0
		for _, cat := range rowCategories {
			catData := data.Categories[cat]
			if len(catData.Codes) > maxCodes {
				maxCodes = len(catData.Codes)
			}
		}
		if maxCodes > maxDetailRows {
			maxCodes = maxDetailRows
		}

		// Detail rows
		for codeIdx := 0; codeIdx < maxCodes; codeIdx++ {
			detailParts := make([]string, len(rowCategories))
			for i, cat := range rowCategories {
				catData := data.Categories[cat]
				if codeIdx < len(catData.Codes) {
					code := catData.Codes[codeIdx]
					detail := fmt.Sprintf("%d: %s (%.1f%%)",
						code.Code,
						formatNumber(code.Count),
						code.Percentage)
					// Apply status color
					styledDetail := StatusCategoryStyle(cat).Render(detail)
					detailParts[i] = padToWidth(styledDetail, colWidth)
				} else {
					detailParts[i] = strings.Repeat(" ", colWidth)
				}
			}
			lines = append(lines, "  "+strings.Join(detailParts, "  "))
		}
	}

	if len(lines) == 0 {
		return "  No status codes"
	}

	return strings.Join(lines, "\n")
}

// padToWidth pads a string to the given width, handling ANSI codes
func padToWidth(s string, width int) string {
	// Get visible width (without ANSI codes)
	visible := stripAnsi(s)
	visibleLen := len(visible)

	if visibleLen >= width {
		return s
	}

	return s + strings.Repeat(" ", width-visibleLen)
}
