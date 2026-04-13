package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOrgSettingsServer returns a test server wired with an in-memory
// orgSettingsRepository alongside the usual auth/mapping stores.
func newOrgSettingsServer(t *testing.T) (*server, string) {
	t.Helper()
	srv, token := newSettingsServer(t)
	srv.orgSettings = &memOrgSettingsRepository{}
	return srv, token
}

// ----------------------------------------------------------------------------
// memOrgSettingsRepository
// ----------------------------------------------------------------------------

func TestMemOrgSettings_DefaultConfig(t *testing.T) {
	repo := &memOrgSettingsRepository{}
	cfg, err := repo.getExportConfig("org-1")
	require.NoError(t, err)
	assert.Equal(t, defaultExportConfig.Columns, cfg.Columns)
	assert.Equal(t, defaultExportConfig.IncludeHeader, cfg.IncludeHeader)
}

func TestMemOrgSettings_SaveAndGet(t *testing.T) {
	repo := &memOrgSettingsRepository{}
	cfg := &ExportConfig{
		Columns:       []string{"internalPartNumber", "quantity", "description"},
		IncludeHeader: true,
	}
	require.NoError(t, repo.saveExportConfig(cfg, "org-1"))

	got, err := repo.getExportConfig("org-1")
	require.NoError(t, err)
	assert.Equal(t, cfg.Columns, got.Columns)
	assert.True(t, got.IncludeHeader)
}

func TestMemOrgSettings_IsolatedPerOrg(t *testing.T) {
	repo := &memOrgSettingsRepository{}
	cfg := &ExportConfig{Columns: []string{"description"}, IncludeHeader: false}
	require.NoError(t, repo.saveExportConfig(cfg, "org-1"))

	// org-2 should still get the default.
	got, err := repo.getExportConfig("org-2")
	require.NoError(t, err)
	assert.Equal(t, defaultExportConfig.Columns, got.Columns)
}

// ----------------------------------------------------------------------------
// GET /api/org/export-config
// ----------------------------------------------------------------------------

func TestGetExportConfig_ReturnsDefault(t *testing.T) {
	srv, token := newOrgSettingsServer(t)
	req := authedRequest(http.MethodGet, "/api/org/export-config", "", token)
	w := httptest.NewRecorder()

	srv.getExportConfig(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var cfg ExportConfig
	require.NoError(t, json.NewDecoder(w.Body).Decode(&cfg))
	assert.Equal(t, defaultExportConfig.Columns, cfg.Columns)
}

// ----------------------------------------------------------------------------
// PUT /api/org/export-config
// ----------------------------------------------------------------------------

func TestSaveExportConfig_SavesAndReturns(t *testing.T) {
	srv, token := newOrgSettingsServer(t)
	body := `{"columns":["internalPartNumber","quantity","unit"],"includeHeader":true}`
	req := authedRequest(http.MethodPut, "/api/org/export-config", body, token)
	w := httptest.NewRecorder()

	srv.saveExportConfig(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var cfg ExportConfig
	require.NoError(t, json.NewDecoder(w.Body).Decode(&cfg))
	assert.Equal(t, []string{"internalPartNumber", "quantity", "unit"}, cfg.Columns)
	assert.True(t, cfg.IncludeHeader)
}

func TestSaveExportConfig_RejectsUnknownColumn(t *testing.T) {
	srv, token := newOrgSettingsServer(t)
	body := `{"columns":["internalPartNumber","bogusColumn"]}`
	req := authedRequest(http.MethodPut, "/api/org/export-config", body, token)
	w := httptest.NewRecorder()

	srv.saveExportConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaveExportConfig_RejectsEmptyColumns(t *testing.T) {
	srv, token := newOrgSettingsServer(t)
	body := `{"columns":[]}`
	req := authedRequest(http.MethodPut, "/api/org/export-config", body, token)
	w := httptest.NewRecorder()

	srv.saveExportConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaveExportConfig_PersistsAcrossCalls(t *testing.T) {
	srv, token := newOrgSettingsServer(t)

	body, _ := json.Marshal(ExportConfig{Columns: []string{"description", "internalPartNumber"}, IncludeHeader: false})
	req := authedRequest(http.MethodPut, "/api/org/export-config", string(body), token)
	_ = httptest.NewRecorder()
	w := httptest.NewRecorder()
	srv.saveExportConfig(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Fetch it back.
	req2 := authedRequest(http.MethodGet, "/api/org/export-config", "", token)
	w2 := httptest.NewRecorder()
	srv.getExportConfig(w2, req2)

	var cfg ExportConfig
	require.NoError(t, json.NewDecoder(w2.Body).Decode(&cfg))
	assert.Equal(t, []string{"description", "internalPartNumber"}, cfg.Columns)
}

// ----------------------------------------------------------------------------
// exportSAP — uses stored config
// ----------------------------------------------------------------------------

func TestExportSAP_UsesStoredConfig(t *testing.T) {
	srv, token := newOrgSettingsServer(t)

	// Set config to output description + internalPartNumber + quantity.
	cfg := ExportConfig{Columns: []string{"description", "internalPartNumber", "quantity"}, IncludeHeader: false}
	cfgBody, _ := json.Marshal(cfg)
	saveReq := authedRequest(http.MethodPut, "/api/org/export-config", string(cfgBody), token)
	srv.saveExportConfig(httptest.NewRecorder(), saveReq)

	// Seed a document.
	doc := &Document{
		ID:       "doc-cfg-1",
		Filename: "test.pdf",
		BOMRows: []BOMRow{
			{
				LineNumber:         1,
				Description:        "Red wire",
				Quantity:           Quantity{Raw: "2", Value: fptr(2.0), Unit: sptr("M")},
				InternalPartNumber: "W-R-2",
			},
		},
	}
	srv.store.save(doc)

	req := authedRequest(http.MethodGet, "/api/documents/doc-cfg-1/export/sap", "", token)
	req.SetPathValue("id", "doc-cfg-1")
	w := httptest.NewRecorder()
	srv.exportSAP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	lines := splitLines(w.Body.Bytes())
	require.Len(t, lines, 1)
	assert.Equal(t, "Red wire\tW-R-2\t2", lines[0])
}

func TestExportSAP_WithHeader(t *testing.T) {
	srv, token := newOrgSettingsServer(t)

	cfg := ExportConfig{Columns: []string{"internalPartNumber", "quantity"}, IncludeHeader: true}
	cfgBody, _ := json.Marshal(cfg)
	saveReq := authedRequest(http.MethodPut, "/api/org/export-config", string(cfgBody), token)
	srv.saveExportConfig(httptest.NewRecorder(), saveReq)

	doc := &Document{
		ID:       "doc-hdr-1",
		Filename: "test.pdf",
		BOMRows: []BOMRow{
			{
				LineNumber:         1,
				Quantity:           Quantity{Raw: "1", Value: fptr(1.0), Unit: sptr("EA")},
				InternalPartNumber: "C-001",
			},
		},
	}
	srv.store.save(doc)

	req := authedRequest(http.MethodGet, "/api/documents/doc-hdr-1/export/sap", "", token)
	req.SetPathValue("id", "doc-hdr-1")
	w := httptest.NewRecorder()
	srv.exportSAP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	lines := splitLines(w.Body.Bytes())
	require.Len(t, lines, 2, "expected header + 1 data row")
	assert.Equal(t, "Internal Part Number\tQuantity", lines[0], "first line should be header")
	assert.Equal(t, "C-001\t1", lines[1])
}

func splitLines(b []byte) []string {
	s := string(bytes.TrimRight(b, "\n"))
	if s == "" {
		return nil
	}
	var parts []string
	for _, p := range bytes.Split([]byte(s), []byte("\n")) {
		parts = append(parts, string(p))
	}
	return parts
}
