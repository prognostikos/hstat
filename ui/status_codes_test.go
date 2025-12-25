package ui

import (
	"strings"
	"testing"

	"github.com/betternow/hstat/store"
)

func TestRenderStatusCodesColumnar_BasicLayout(t *testing.T) {
	// Create test data with various status codes
	data := StatusCodesData{
		Categories: map[int]CategoryData{
			2: {
				Total:      100,
				Percentage: 80.0,
				Codes: []CodeData{
					{Code: 200, Count: 90, Percentage: 72.0},
					{Code: 201, Count: 10, Percentage: 8.0},
				},
			},
			4: {
				Total:      20,
				Percentage: 16.0,
				Codes: []CodeData{
					{Code: 404, Count: 15, Percentage: 12.0},
					{Code: 400, Count: 5, Percentage: 4.0},
				},
			},
			5: {
				Total:      5,
				Percentage: 4.0,
				Codes: []CodeData{
					{Code: 500, Count: 5, Percentage: 4.0},
				},
			},
		},
	}

	result := RenderStatusCodesColumnar(data, 120, 5)

	// Should contain category headers with percentages
	if !strings.Contains(result, "2xx") {
		t.Error("expected result to contain '2xx'")
	}
	if !strings.Contains(result, "4xx") {
		t.Error("expected result to contain '4xx'")
	}
	if !strings.Contains(result, "5xx") {
		t.Error("expected result to contain '5xx'")
	}

	// Should contain individual codes
	if !strings.Contains(result, "200") {
		t.Error("expected result to contain '200'")
	}
	if !strings.Contains(result, "404") {
		t.Error("expected result to contain '404'")
	}
	if !strings.Contains(result, "500") {
		t.Error("expected result to contain '500'")
	}
}

func TestRenderStatusCodesColumnar_EmptyData(t *testing.T) {
	data := StatusCodesData{
		Categories: map[int]CategoryData{},
	}

	result := RenderStatusCodesColumnar(data, 120, 5)

	// Should not crash and should return something
	if result == "" {
		t.Error("expected non-empty result even with no data")
	}
}

func TestRenderStatusCodesColumnar_SingleCategory(t *testing.T) {
	data := StatusCodesData{
		Categories: map[int]CategoryData{
			2: {
				Total:      100,
				Percentage: 100.0,
				Codes: []CodeData{
					{Code: 200, Count: 100, Percentage: 100.0},
				},
			},
		},
	}

	result := RenderStatusCodesColumnar(data, 120, 5)

	// Should only show 2xx
	if !strings.Contains(result, "2xx") {
		t.Error("expected result to contain '2xx'")
	}
	if !strings.Contains(result, "200") {
		t.Error("expected result to contain '200'")
	}
}

func TestRenderStatusCodesColumnar_NarrowWidth(t *testing.T) {
	data := StatusCodesData{
		Categories: map[int]CategoryData{
			2: {Total: 80, Percentage: 80.0, Codes: []CodeData{{Code: 200, Count: 80, Percentage: 80.0}}},
			3: {Total: 10, Percentage: 10.0, Codes: []CodeData{{Code: 301, Count: 10, Percentage: 10.0}}},
			4: {Total: 5, Percentage: 5.0, Codes: []CodeData{{Code: 404, Count: 5, Percentage: 5.0}}},
			5: {Total: 5, Percentage: 5.0, Codes: []CodeData{{Code: 500, Count: 5, Percentage: 5.0}}},
		},
	}

	// Narrow width should still render all categories (may wrap)
	result := RenderStatusCodesColumnar(data, 60, 3)

	// Should still contain all categories
	if !strings.Contains(result, "2xx") {
		t.Error("expected narrow result to contain '2xx'")
	}
	if !strings.Contains(result, "4xx") {
		t.Error("expected narrow result to contain '4xx'")
	}
}

func TestRenderStatusCodesColumnar_PercentageFormat(t *testing.T) {
	data := StatusCodesData{
		Categories: map[int]CategoryData{
			2: {
				Total:      100,
				Percentage: 85.5,
				Codes:      []CodeData{},
			},
		},
	}

	result := RenderStatusCodesColumnar(data, 120, 5)

	// Should show percentage in header
	if !strings.Contains(result, "85.5%") && !strings.Contains(result, "85.5") {
		t.Error("expected result to contain percentage '85.5'")
	}
}

func TestRenderStatusCodesColumnar_ZeroPercentage(t *testing.T) {
	data := StatusCodesData{
		Categories: map[int]CategoryData{
			1: {
				Total:      0,
				Percentage: 0.0,
				Codes:      []CodeData{},
			},
			2: {
				Total:      100,
				Percentage: 100.0,
				Codes:      []CodeData{{Code: 200, Count: 100, Percentage: 100.0}},
			},
		},
	}

	result := RenderStatusCodesColumnar(data, 120, 5)

	// Should show dash for zero percentage
	if !strings.Contains(result, "1xx") {
		t.Error("expected result to contain '1xx' even with zero count")
	}
}

func TestStatusCodesDataFromStore(t *testing.T) {
	// Test converting store data to StatusCodesData
	storeCounts := []store.StatusCountItem{
		{Status: 200, Count: 80},
		{Status: 201, Count: 10},
		{Status: 404, Count: 5},
		{Status: 500, Count: 5},
	}

	data := StatusCodesDataFromStore(storeCounts)

	// Check category 2xx
	cat2, ok := data.Categories[2]
	if !ok {
		t.Fatal("expected category 2 to exist")
	}
	if cat2.Total != 90 {
		t.Errorf("expected 2xx total 90, got %d", cat2.Total)
	}

	// Check category 4xx
	cat4, ok := data.Categories[4]
	if !ok {
		t.Fatal("expected category 4 to exist")
	}
	if cat4.Total != 5 {
		t.Errorf("expected 4xx total 5, got %d", cat4.Total)
	}

	// Check category 5xx
	cat5, ok := data.Categories[5]
	if !ok {
		t.Fatal("expected category 5 to exist")
	}
	if cat5.Total != 5 {
		t.Errorf("expected 5xx total 5, got %d", cat5.Total)
	}
}

func TestCalculateStatusCodesColumns(t *testing.T) {
	// Wide terminal should fit 5 columns
	cols := calculateStatusCodeColumns(150)
	if cols != 5 {
		t.Errorf("expected 5 columns for width 150, got %d", cols)
	}

	// Medium terminal should fit fewer
	cols = calculateStatusCodeColumns(80)
	if cols < 3 || cols > 5 {
		t.Errorf("expected 3-5 columns for width 80, got %d", cols)
	}

	// Very narrow should have at least 1
	cols = calculateStatusCodeColumns(40)
	if cols < 1 {
		t.Errorf("expected at least 1 column for width 40, got %d", cols)
	}
}

func TestRenderStatusCodesColumnar_MaxDetailRows(t *testing.T) {
	// When there are many codes in a category, limit the detail rows
	data := StatusCodesData{
		Categories: map[int]CategoryData{
			2: {
				Total:      100,
				Percentage: 100.0,
				Codes: []CodeData{
					{Code: 200, Count: 50, Percentage: 50.0},
					{Code: 201, Count: 20, Percentage: 20.0},
					{Code: 202, Count: 10, Percentage: 10.0},
					{Code: 203, Count: 10, Percentage: 10.0},
					{Code: 204, Count: 5, Percentage: 5.0},
					{Code: 206, Count: 5, Percentage: 5.0},
				},
			},
		},
	}

	// With maxRows=3, should show top 3 codes
	result := RenderStatusCodesColumnar(data, 120, 3)

	// Should contain top codes
	if !strings.Contains(result, "200") {
		t.Error("expected result to contain '200'")
	}
	if !strings.Contains(result, "201") {
		t.Error("expected result to contain '201'")
	}
}
