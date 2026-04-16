package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers ─────────────────────────────────────────────────────────────────────

func makeDoc(id, filename string, status DocumentStatus, cpns ...string) *Document {
	rows := make([]BOMRow, len(cpns))
	for i, cpn := range cpns {
		rows[i] = BOMRow{
			ID:                 fmt.Sprintf("row-%d", i+1),
			LineNumber:         i + 1,
			CustomerPartNumber: cpn,
			Quantity:           Quantity{Raw: "1", Flags: []string{}},
			Flags:              []string{},
		}
	}
	return &Document{
		ID:         id,
		Filename:   filename,
		Status:     status,
		UploadedAt: time.Now().UTC(),
		BOMRows:    rows,
		Warnings:   []string{},
	}
}

// ── rankSimilarDocuments ──────────────────────────────────────────────────────

func TestRankSimilarDocuments_EmptyCandidates(t *testing.T) {
	doc := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1")
	results := rankSimilarDocuments(doc, nil, 0)
	assert.Empty(t, results)
}

func TestRankSimilarDocuments_SelfExcluded(t *testing.T) {
	doc := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1")
	results := rankSimilarDocuments(doc, []*Document{doc}, 0)
	assert.Empty(t, results)
}

func TestRankSimilarDocuments_SkipsNonDone(t *testing.T) {
	query := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1")
	uploading := makeDoc("b", "harness-001.pdf", StatusUploaded, "CPN-1")
	analyzing := makeDoc("c", "harness-001.pdf", StatusAnalyzing, "CPN-1")
	errored := makeDoc("d", "harness-001.pdf", StatusError, "CPN-1")
	results := rankSimilarDocuments(query, []*Document{uploading, analyzing, errored}, 0)
	assert.Empty(t, results)
}

func TestRankSimilarDocuments_SkipsEmptyBOM(t *testing.T) {
	query := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1")
	empty := makeDoc("b", "harness-001.pdf", StatusDone) // no CPNs → no rows
	results := rankSimilarDocuments(query, []*Document{empty}, 0)
	assert.Empty(t, results)
}

func TestRankSimilarDocuments_FilenameTokenOverlap(t *testing.T) {
	query := makeDoc("a", "harness-001-revA.pdf", StatusDone, "CPN-A")
	past := makeDoc("b", "harness-001-revB.pdf", StatusDone, "CPN-B") // diff CPN but same filename base
	results := rankSimilarDocuments(query, []*Document{past}, 0)
	require.Len(t, results, 1)
	assert.Greater(t, results[0].Score, 0.0)
	assert.NotEmpty(t, results[0].MatchReasons)
}

func TestRankSimilarDocuments_SharedCPNs(t *testing.T) {
	query := makeDoc("a", "drawing-x.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-3")
	past := makeDoc("b", "drawing-y.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-9")
	results := rankSimilarDocuments(query, []*Document{past}, 0)
	require.Len(t, results, 1)
	assert.Greater(t, results[0].Score, 0.0)
}

func TestRankSimilarDocuments_SharedMPNs(t *testing.T) {
	query := &Document{
		ID: "a", Filename: "drawing-q.pdf", Status: StatusDone,
		BOMRows: []BOMRow{
			{ID: "r1", ManufacturerPartNumber: "MPN-XYZ", Quantity: Quantity{Raw: "1", Flags: []string{}}, Flags: []string{}},
		},
	}
	past := &Document{
		ID: "b", Filename: "drawing-p.pdf", Status: StatusDone,
		BOMRows: []BOMRow{
			{ID: "r1", ManufacturerPartNumber: "MPN-XYZ", Quantity: Quantity{Raw: "1", Flags: []string{}}, Flags: []string{}},
		},
	}
	results := rankSimilarDocuments(query, []*Document{past}, 0)
	require.Len(t, results, 1)
	assert.Greater(t, results[0].Score, 0.0)
}

func TestRankSimilarDocuments_Ranking(t *testing.T) {
	query := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-3")
	// strong: filename match + 2 shared CPNs
	strong := makeDoc("b", "harness-001-v2.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-Z")
	// weak: filename mismatch + 1 shared CPN (score ~0.10, below default threshold)
	weak := makeDoc("c", "completely-different.pdf", StatusDone, "CPN-1", "CPN-X", "CPN-Y")

	// With threshold=0 both are shown; strong must rank first.
	results := rankSimilarDocuments(query, []*Document{weak, strong}, 0)
	require.Len(t, results, 2)
	assert.Equal(t, "b", results[0].ID, "strong match should rank first")
	assert.Equal(t, "c", results[1].ID, "weak match should rank second")
}

func TestRankSimilarDocuments_MaxFive(t *testing.T) {
	query := makeDoc("q", "drawing.pdf", StatusDone, "CPN-1")
	var candidates []*Document
	for i := 0; i < 10; i++ {
		candidates = append(candidates, makeDoc(
			fmt.Sprintf("doc-%d", i),
			fmt.Sprintf("drawing-%d.pdf", i),
			StatusDone,
			"CPN-1",
		))
	}
	results := rankSimilarDocuments(query, candidates, 0)
	assert.LessOrEqual(t, len(results), 5)
}

func TestRankSimilarDocuments_Fields(t *testing.T) {
	query := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1")
	past := makeDoc("b", "harness-001.pdf", StatusDone, "CPN-1")
	past.UploadedAt = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	results := rankSimilarDocuments(query, []*Document{past}, 0)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, "b", r.ID)
	assert.Equal(t, "harness-001.pdf", r.Filename)
	assert.Equal(t, 1, r.BOMRowCount)
	assert.Equal(t, past.UploadedAt, r.UploadedAt)
}

// ── threshold filtering ───────────────────────────────────────────────────────

func TestRankSimilarDocuments_ThresholdExcludesWeak(t *testing.T) {
	query := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-3")
	// weak doc: 1 shared CPN out of 5 total → Jaccard=0.2 → score=0.10 (below 0.15)
	weak := makeDoc("b", "completely-different.pdf", StatusDone, "CPN-1", "CPN-X", "CPN-Y")

	results := rankSimilarDocuments(query, []*Document{weak}, 0.15)
	assert.Empty(t, results, "candidate with score below threshold should be excluded")
}

func TestRankSimilarDocuments_ThresholdPassesStrong(t *testing.T) {
	query := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-3")
	strong := makeDoc("b", "harness-001-v2.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-Z")

	results := rankSimilarDocuments(query, []*Document{strong}, 0.15)
	require.Len(t, results, 1, "candidate above threshold should be included")
	assert.Equal(t, "b", results[0].ID)
}

func TestRankSimilarDocuments_ThresholdZeroShowsAll(t *testing.T) {
	query := makeDoc("a", "harness-001.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-3")
	weak := makeDoc("b", "completely-different.pdf", StatusDone, "CPN-1", "CPN-X", "CPN-Y")
	strong := makeDoc("c", "harness-001-v2.pdf", StatusDone, "CPN-1", "CPN-2", "CPN-Z")

	results := rankSimilarDocuments(query, []*Document{weak, strong}, 0)
	assert.Len(t, results, 2, "threshold=0 should not filter anything with score>0")
}

// ── score breakdown ───────────────────────────────────────────────────────────

func TestRankSimilarDocuments_BreakdownFilenameOnly(t *testing.T) {
	// Same filename base, no shared part numbers.
	query := makeDoc("a", "harness-001-revA.pdf", StatusDone, "CPN-A")
	past := makeDoc("b", "harness-001-revB.pdf", StatusDone, "CPN-B")

	results := rankSimilarDocuments(query, []*Document{past}, 0)
	require.Len(t, results, 1)
	bd := results[0].ScoreBreakdown
	assert.Greater(t, bd.Filename, 0.0, "filename score should be populated")
	assert.Equal(t, 0.0, bd.CPN, "CPN score should be zero")
	assert.Equal(t, 0.0, bd.MPN, "MPN score should be zero")
}

func TestRankSimilarDocuments_BreakdownCPNOnly(t *testing.T) {
	// No filename overlap, shared CPNs.
	query := makeDoc("a", "drawing-x.pdf", StatusDone, "CPN-1", "CPN-2")
	past := makeDoc("b", "drawing-y.pdf", StatusDone, "CPN-1", "CPN-2")

	results := rankSimilarDocuments(query, []*Document{past}, 0)
	require.Len(t, results, 1)
	bd := results[0].ScoreBreakdown
	assert.Greater(t, bd.CPN, 0.0, "CPN score should be populated")
}

// ── filenameTokens ────────────────────────────────────────────────────────────

func TestFilenameTokens(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect []string
	}{
		{"strips extension", "harness.pdf", []string{"harness"}},
		{"splits on dash", "harness-001.pdf", []string{"harness", "001"}},
		{"splits on underscore", "harness_001.pdf", []string{"harness", "001"}},
		{"lowercases", "HarnessRev2.pdf", []string{"harnessrev2"}},
		{"skips single chars", "a-b-harness.pdf", []string{"harness"}},
		{"empty", ".pdf", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filenameTokens(tc.input)
			for _, want := range tc.expect {
				assert.True(t, got[want], "expected token %q in %v", want, got)
			}
		})
	}
}

// ── jaccardSimilarity ─────────────────────────────────────────────────────────

func TestJaccardSimilarity(t *testing.T) {
	set := func(vs ...string) map[string]bool {
		m := make(map[string]bool)
		for _, v := range vs {
			m[v] = true
		}
		return m
	}

	assert.Equal(t, 0.0, jaccardSimilarity(nil, nil))
	assert.Equal(t, 1.0, jaccardSimilarity(set("a", "b"), set("a", "b")))
	assert.Equal(t, 0.0, jaccardSimilarity(set("a"), set("b")))
	assert.InDelta(t, 1.0/3.0, jaccardSimilarity(set("a", "b"), set("a", "c")), 0.001)
}
