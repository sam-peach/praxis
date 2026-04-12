package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------------
// parseQuantity
// ----------------------------------------------------------------------------

func TestParseQuantity_PlainInteger(t *testing.T) {
	q := parseQuantity("4", "EA")

	assert.Equal(t, "4", q.Raw)
	require.NotNil(t, q.Value)
	assert.Equal(t, 4.0, *q.Value)
	require.NotNil(t, q.Unit)
	assert.Equal(t, "EA", *q.Unit)
	assert.Empty(t, q.Flags)
}

func TestParseQuantity_PlainDecimal(t *testing.T) {
	q := parseQuantity("0.35", "M")

	require.NotNil(t, q.Value)
	assert.Equal(t, 0.35, *q.Value)
	assert.Empty(t, q.Flags)
}

func TestParseQuantity_InlineUnitMatchesDeclared(t *testing.T) {
	// "0.35m" with declared unit "M" — compatible, no flag.
	q := parseQuantity("0.35m", "M")

	require.NotNil(t, q.Value)
	assert.Equal(t, 0.35, *q.Value)
	assert.Empty(t, q.Flags, "compatible units should not be flagged")
}

func TestParseQuantity_InlineUnitConflictsDeclared(t *testing.T) {
	// Core bug case: "150mm" with declared unit "M" — these are NOT compatible.
	// Raw value must be preserved; unit_ambiguous must be set.
	q := parseQuantity("150mm", "M")

	assert.Equal(t, "150mm", q.Raw, "Raw must never be transformed")
	require.NotNil(t, q.Value)
	assert.Equal(t, 150.0, *q.Value, "Value should be 150, not 0.15")
	require.NotNil(t, q.Unit)
	assert.Equal(t, "MM", *q.Unit, "Unit should be the inline unit from the drawing")
	assert.Contains(t, q.Flags, "unit_ambiguous")
}

func TestParseQuantity_Unparseable(t *testing.T) {
	q := parseQuantity("approx 5", "EA")

	assert.Equal(t, "approx 5", q.Raw)
	assert.Nil(t, q.Value)
	assert.Contains(t, q.Flags, "unit_ambiguous")
}

func TestParseQuantity_EmptyRaw(t *testing.T) {
	q := parseQuantity("", "EA")

	assert.Equal(t, "", q.Raw)
	assert.Nil(t, q.Value)
	assert.Empty(t, q.Flags)
}

func TestParseQuantity_NormalizedEqualsValue(t *testing.T) {
	// Normalized should equal Value — we do not silently convert units.
	q := parseQuantity("3", "EA")

	require.NotNil(t, q.Value)
	require.NotNil(t, q.Normalized)
	assert.Equal(t, *q.Value, *q.Normalized)
}

// ----------------------------------------------------------------------------
// unitCompatible
// ----------------------------------------------------------------------------

func TestUnitCompatible(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"M", "M", true},
		{"M", "METRES", true},
		{"METER", "M", true},
		{"MM", "M", false},   // millimetres ≠ metres
		{"EA", "EACH", true},
		{"EA", "M", false},
		{"MM", "MILLIMETRES", true},
		{"KG", "KILOGRAMS", true},
		{"KG", "EA", false},
		{"", "", false},
	}
	for _, tc := range tests {
		got := unitCompatible(tc.a, tc.b)
		assert.Equal(t, tc.want, got, "unitCompatible(%q, %q)", tc.a, tc.b)
	}
}

// ----------------------------------------------------------------------------
// detectSupplier
// ----------------------------------------------------------------------------

func TestDetectSupplier_RS(t *testing.T) {
	row := &BOMRow{SupplierReference: "679-4698", ManufacturerPartNumber: "some-mpn"}
	detectSupplier(row)

	assert.Equal(t, "RS", row.Supplier)
	assert.Contains(t, row.Flags, "supplier_reference_detected")
}

func TestDetectSupplier_RSSevenDigit(t *testing.T) {
	row := &BOMRow{SupplierReference: "1234567", ManufacturerPartNumber: "some-mpn"}
	detectSupplier(row)

	assert.Equal(t, "RS", row.Supplier)
}

func TestDetectSupplier_Farnell(t *testing.T) {
	row := &BOMRow{SupplierReference: "1234567A", ManufacturerPartNumber: "some-mpn"}
	detectSupplier(row)

	assert.Equal(t, "Farnell", row.Supplier)
}

func TestDetectSupplier_Unknown(t *testing.T) {
	row := &BOMRow{SupplierReference: "DIGIKEY-ABC123", ManufacturerPartNumber: "some-mpn"}
	detectSupplier(row)

	assert.Equal(t, "Unknown", row.Supplier)
	assert.Contains(t, row.Flags, "supplier_reference_detected")
}

func TestDetectSupplier_NoRef(t *testing.T) {
	row := &BOMRow{SupplierReference: ""}
	detectSupplier(row)

	assert.Equal(t, "", row.Supplier)
	assert.NotContains(t, row.Flags, "supplier_reference_detected")
}

func TestDetectSupplier_MissingMPN_AddsNote(t *testing.T) {
	row := &BOMRow{SupplierReference: "679-4698", ManufacturerPartNumber: ""}
	detectSupplier(row)

	assert.Contains(t, row.Notes, "Supplier reference detected")
}

// ----------------------------------------------------------------------------
// enrichFromSupplierRef
// ----------------------------------------------------------------------------

func TestEnrichFromSupplierRef_AddsMockMPN(t *testing.T) {
	row := &BOMRow{SupplierReference: "679-4698", ManufacturerPartNumber: "", Confidence: 0.9}
	enrichFromSupplierRef(row)

	assert.NotEmpty(t, row.ManufacturerPartNumber)
	assert.Contains(t, row.Notes, "verify before use")
	assert.Contains(t, row.Flags, "low_confidence")
	assert.LessOrEqual(t, row.Confidence, 0.6)
}

func TestEnrichFromSupplierRef_SkipsIfMPNExists(t *testing.T) {
	row := &BOMRow{SupplierReference: "679-4698", ManufacturerPartNumber: "22-01-3067"}
	enrichFromSupplierRef(row)

	assert.Equal(t, "22-01-3067", row.ManufacturerPartNumber, "existing MPN must not be overwritten")
}

func TestEnrichFromSupplierRef_SkipsIfNoRef(t *testing.T) {
	row := &BOMRow{SupplierReference: "", ManufacturerPartNumber: ""}
	enrichFromSupplierRef(row)

	assert.Empty(t, row.ManufacturerPartNumber)
}

// ----------------------------------------------------------------------------
// appendFlag / appendNote
// ----------------------------------------------------------------------------

func TestAppendFlag_NoDuplicates(t *testing.T) {
	flags := []string{"a", "b"}
	flags = appendFlag(flags, "b")
	assert.Equal(t, []string{"a", "b"}, flags)
}

func TestAppendFlag_AddsNew(t *testing.T) {
	flags := appendFlag([]string{"a"}, "b")
	assert.Equal(t, []string{"a", "b"}, flags)
}

func TestAppendNote_Empty(t *testing.T) {
	assert.Equal(t, "new note", appendNote("", "new note"))
}

func TestAppendNote_Append(t *testing.T) {
	assert.Equal(t, "first; second", appendNote("first", "second"))
}

// ----------------------------------------------------------------------------
// applyMapping
// ----------------------------------------------------------------------------

func TestApplyMapping_FillsFields(t *testing.T) {
	ms := newMappingStoreInMemory()
	_ = ms.save(&Mapping{
		CustomerPartNumber:     "CUST-CT-01",
		InternalPartNumber:     "SC-001",
		ManufacturerPartNumber: "MPN-001",
	})

	row := &BOMRow{CustomerPartNumber: "CUST-CT-01"}
	applyMapping(row, ms)

	assert.Equal(t, "SC-001", row.InternalPartNumber)
	assert.Equal(t, "MPN-001", row.ManufacturerPartNumber)
	assert.Contains(t, row.Flags, "mapping_applied")
	assert.Contains(t, row.Notes, "Matched from previous mapping")
}

func TestApplyMapping_CaseInsensitive(t *testing.T) {
	ms := newMappingStoreInMemory()
	_ = ms.save(&Mapping{CustomerPartNumber: "cust-ct-01", InternalPartNumber: "SC-001"})

	row := &BOMRow{CustomerPartNumber: "CUST-CT-01"}
	applyMapping(row, ms)

	assert.Equal(t, "SC-001", row.InternalPartNumber)
}

func TestApplyMapping_NoMatch(t *testing.T) {
	ms := newMappingStoreInMemory()
	row := &BOMRow{CustomerPartNumber: "UNKNOWN"}
	applyMapping(row, ms)

	assert.Empty(t, row.InternalPartNumber)
	assert.NotContains(t, row.Flags, "mapping_applied")
}

func TestApplyMapping_DoesNotOverwriteExisting(t *testing.T) {
	ms := newMappingStoreInMemory()
	_ = ms.save(&Mapping{CustomerPartNumber: "CUST-CT-01", InternalPartNumber: "SC-001"})

	row := &BOMRow{
		CustomerPartNumber: "CUST-CT-01",
		InternalPartNumber: "SC-ALREADY-SET",
	}
	applyMapping(row, ms)

	assert.Equal(t, "SC-ALREADY-SET", row.InternalPartNumber, "existing internal PN must not be overwritten")
}

// newMappingStoreInMemory creates an in-memory store with no file backing — for tests only.
func newMappingStoreInMemory() *mappingStore {
	return &mappingStore{data: make(map[string]*Mapping), filePath: ""}
}
