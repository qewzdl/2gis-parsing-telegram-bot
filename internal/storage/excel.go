package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"
	"github.com/yourusername/2gis-parser/internal/models"
)

// ExportToExcel creates an XLSX file from a list of companies.
func ExportToExcel(companies []models.Company, outputDir, query string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	f := excelize.NewFile()
	sheet := "Companies"
	f.SetSheetName("Sheet1", sheet)

	// Header style.
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1E3A5F"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "4472C4", Style: 2},
		},
	})

	// Even row style.
	evenStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"EBF2FF"}, Pattern: 1},
	})

	headers := []string{"No.", "Name", "City", "Address", "Phone", "Website", "Category", "Coordinates"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	widths := []float64{5, 40, 15, 45, 18, 30, 25, 25}

	// Headers.
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", cols[i])
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
		f.SetColWidth(sheet, cols[i], cols[i], widths[i])
	}
	f.SetRowHeight(sheet, 1, 22)

	// Data.
	for idx, c := range companies {
		row := idx + 2
		coords := ""
		if c.Lat != 0 {
			coords = fmt.Sprintf("%.6f, %.6f", c.Lat, c.Lon)
		}

		values := []interface{}{
			idx + 1,
			c.Name,
			c.City,
			c.Address,
			c.Phone,
			c.Website,
			c.Category,
			coords,
		}

		for i, v := range values {
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], row), v)
		}

		// Alternating rows.
		if idx%2 == 1 {
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), evenStyle)
		}
	}

	// Auto filter.
	if err := f.AutoFilter(sheet, fmt.Sprintf("A1:H%d", len(companies)+1), nil); err != nil {
		return "", fmt.Errorf("failed to add auto filter: %w", err)
	}

	// Freeze the first row.
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Summary row.
	lastRow := len(companies) + 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", lastRow), fmt.Sprintf("Total: %d companies", len(companies)))
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Italic: true, Color: "1E3A5F"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", lastRow), fmt.Sprintf("H%d", lastRow), totalStyle)

	// File name.
	ts := time.Now().Format("02-01-2006_15-04")
	filename := fmt.Sprintf("2gis_%s_%s.xlsx", sanitize(query), ts)
	path := filepath.Join(outputDir, filename)

	if err := f.SaveAs(path); err != nil {
		return "", fmt.Errorf("saving file: %w", err)
	}

	return path, nil
}

func sanitize(s string) string {
	result := []rune{}
	for _, r := range s {
		if r == ' ' {
			result = append(result, '_')
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, r)
		}
	}
	if len(result) > 30 {
		result = result[:30]
	}
	return string(result)
}
