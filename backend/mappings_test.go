package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMappingStore_SaveAndLookup(t *testing.T) {
	ms := newMappingStoreInMemory()

	m := &Mapping{
		CustomerPartNumber:     "CUST-001",
		InternalPartNumber:     "SC-001",
		ManufacturerPartNumber: "MPN-001",
		Description:            "Test part",
		Source:                 "manual",
	}
	require.NoError(t, ms.save(m))

	got, ok := ms.lookup("CUST-001")
	require.True(t, ok)
	assert.Equal(t, "SC-001", got.InternalPartNumber)
	assert.Equal(t, "MPN-001", got.ManufacturerPartNumber)
}

func TestMappingStore_LookupCaseInsensitive(t *testing.T) {
	ms := newMappingStoreInMemory()
	require.NoError(t, ms.save(&Mapping{CustomerPartNumber: "cust-001", InternalPartNumber: "SC-001"}))

	_, ok := ms.lookup("CUST-001")
	assert.True(t, ok)

	_, ok = ms.lookup("Cust-001")
	assert.True(t, ok)
}

func TestMappingStore_LookupMiss(t *testing.T) {
	ms := newMappingStoreInMemory()
	_, ok := ms.lookup("MISSING")
	assert.False(t, ok)
}

func TestMappingStore_LookupEmptyKey(t *testing.T) {
	ms := newMappingStoreInMemory()
	_, ok := ms.lookup("")
	assert.False(t, ok)
}

func TestMappingStore_SaveAssignsID(t *testing.T) {
	ms := newMappingStoreInMemory()
	m := &Mapping{CustomerPartNumber: "CUST-001"}
	require.NoError(t, ms.save(m))

	assert.NotEmpty(t, m.ID)
}

func TestMappingStore_SaveSetsTimestamps(t *testing.T) {
	before := time.Now()
	ms := newMappingStoreInMemory()
	m := &Mapping{CustomerPartNumber: "CUST-001"}
	require.NoError(t, ms.save(m))

	assert.True(t, m.CreatedAt.After(before) || m.CreatedAt.Equal(before))
	assert.True(t, m.UpdatedAt.After(before) || m.UpdatedAt.Equal(before))
}

func TestMappingStore_UpdatePreservesCreatedAt(t *testing.T) {
	ms := newMappingStoreInMemory()
	m := &Mapping{CustomerPartNumber: "CUST-001", InternalPartNumber: "SC-001"}
	require.NoError(t, ms.save(m))
	created := m.CreatedAt

	// Update the same key.
	m2 := &Mapping{CustomerPartNumber: "CUST-001", InternalPartNumber: "SC-002"}
	require.NoError(t, ms.save(m2))

	got, _ := ms.lookup("CUST-001")
	assert.Equal(t, created, got.CreatedAt, "CreatedAt must not change on update")
	assert.Equal(t, "SC-002", got.InternalPartNumber)
}

func TestMappingStore_SaveRequiresCustomerPartNumber(t *testing.T) {
	ms := newMappingStoreInMemory()
	err := ms.save(&Mapping{CustomerPartNumber: ""})
	assert.Error(t, err)
}

func TestMappingStore_All(t *testing.T) {
	ms := newMappingStoreInMemory()
	require.NoError(t, ms.save(&Mapping{CustomerPartNumber: "CUST-001"}))
	require.NoError(t, ms.save(&Mapping{CustomerPartNumber: "CUST-002"}))

	all := ms.all()
	assert.Len(t, all, 2)
}

func TestMappingStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mappings.json")

	// Write via one store instance.
	ms1, err := newMappingStore(path)
	require.NoError(t, err)
	require.NoError(t, ms1.save(&Mapping{
		CustomerPartNumber:     "CUST-001",
		InternalPartNumber:     "SC-001",
		ManufacturerPartNumber: "MPN-001",
	}))

	// Read it back via a fresh instance.
	ms2, err := newMappingStore(path)
	require.NoError(t, err)

	got, ok := ms2.lookup("CUST-001")
	require.True(t, ok)
	assert.Equal(t, "SC-001", got.InternalPartNumber)
	assert.Equal(t, "MPN-001", got.ManufacturerPartNumber)
}

func TestMappingStore_MissingFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json")

	ms, err := newMappingStore(path)
	require.NoError(t, err)
	assert.Empty(t, ms.all())
}

func TestMappingStore_CorruptFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mappings.json")
	require.NoError(t, os.WriteFile(path, []byte("not json {{{"), 0644))

	_, err := newMappingStore(path)
	assert.Error(t, err)
}
