package main

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newExportServer(t *testing.T, rows []BOMRow) (*server, string) {
	t.Helper()
	srv, token := newSettingsServer(t)
	doc := &Document{
		ID:       "doc-1",
		Filename: "harness-drawing.pdf",
		BOMRows:  rows,
	}
	srv.store.save(doc)
	return srv, token
}

func fptr(f float64) *float64 { return &f }
func sptr(s string) *string   { return &s }

// TestExportCSV_QuantityIsNumericValue verifies that the Qty column in the
// CSV contains the parsed numeric Value, not the raw drawing string.
func TestExportCSV_QuantityIsNumericValue(t *testing.T) {
	rows := []BOMRow{
		{
			ID:         "r1",
			LineNumber: 1,
			Description: "Red wire",
			Quantity: Quantity{
				Raw:   "150mm",
				Value: fptr(150),
				Unit:  sptr("MM"),
			},
		},
	}
	srv, token := newExportServer(t, rows)
	req := authedRequest(http.MethodGet, "/api/documents/doc-1/bom.csv", "", token)
	req.SetPathValue("id", "doc-1")
	w := httptest.NewRecorder()

	srv.exportCSV(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	records, err := csv.NewReader(w.Body).ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("expected header + 1 data row, got %d rows", len(records))
	}

	// Find Quantity column index from header
	header := records[0]
	qtyIdx := -1
	for i, h := range header {
		if h == "Quantity" {
			qtyIdx = i
			break
		}
	}
	if qtyIdx < 0 {
		t.Fatal("Quantity column not found in CSV header")
	}

	got := records[1][qtyIdx]
	if got != "150" {
		t.Errorf("Quantity column: want %q, got %q (should be parsed Value, not raw %q)", "150", got, "150mm")
	}
}

// TestExportCSV_QuantityFallsBackToRaw verifies that when Value is nil
// the raw string is used as a fallback.
func TestExportCSV_QuantityFallsBackToRaw(t *testing.T) {
	rows := []BOMRow{
		{
			ID:         "r1",
			LineNumber: 1,
			Description: "Unknown qty item",
			Quantity: Quantity{
				Raw:   "TBC",
				Value: nil,
				Unit:  nil,
			},
		},
	}
	srv, token := newExportServer(t, rows)
	req := authedRequest(http.MethodGet, "/api/documents/doc-1/bom.csv", "", token)
	req.SetPathValue("id", "doc-1")
	w := httptest.NewRecorder()

	srv.exportCSV(w, req)

	records, _ := csv.NewReader(w.Body).ReadAll()
	header := records[0]
	qtyIdx := -1
	for i, h := range header {
		if h == "Quantity" {
			qtyIdx = i
			break
		}
	}
	got := records[1][qtyIdx]
	if got != "TBC" {
		t.Errorf("Quantity column: want raw fallback %q, got %q", "TBC", got)
	}
}

// TestExportTSV_TabSeparated verifies that ?format=tsv produces a
// tab-delimited file with the correct Content-Type.
func TestExportTSV_TabSeparated(t *testing.T) {
	rows := []BOMRow{
		{
			ID:         "r1",
			LineNumber: 1,
			Description: "Black wire",
			Quantity: Quantity{
				Raw:   "2",
				Value: fptr(2),
				Unit:  sptr("M"),
			},
			CustomerPartNumber:     "C-001",
			InternalPartNumber:     "I-001",
			ManufacturerPartNumber: "M-001",
		},
	}
	srv, token := newExportServer(t, rows)
	req := authedRequest(http.MethodGet, "/api/documents/doc-1/bom.csv?format=tsv", "", token)
	req.SetPathValue("id", "doc-1")
	w := httptest.NewRecorder()

	srv.exportCSV(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/tab-separated-values") {
		t.Errorf("Content-Type: want text/tab-separated-values, got %q", ct)
	}

	body := w.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + 1 data row, got %d lines", len(lines))
	}
	// Header must use tabs
	if !strings.Contains(lines[0], "\t") {
		t.Errorf("TSV header should contain tab characters: %q", lines[0])
	}
	// Data row should contain numeric qty
	if !strings.Contains(lines[1], "2") {
		t.Errorf("TSV data row should contain numeric quantity: %q", lines[1])
	}
}

// TestExportTSV_ContentDisposition verifies the filename suffix for TSV.
func TestExportTSV_ContentDisposition(t *testing.T) {
	srv, token := newExportServer(t, nil)
	req := authedRequest(http.MethodGet, "/api/documents/doc-1/bom.csv?format=tsv", "", token)
	req.SetPathValue("id", "doc-1")
	w := httptest.NewRecorder()

	srv.exportCSV(w, req)

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".tsv") {
		t.Errorf("Content-Disposition should reference .tsv file, got %q", cd)
	}
}
