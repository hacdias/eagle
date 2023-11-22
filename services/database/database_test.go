package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatabase(t *testing.T) {
	tempDir := t.TempDir()
	db, err := NewDatabase(filepath.Join(tempDir, "bolt.db"))
	require.NoError(t, err)

	err = db.AddTaxonomy(context.Background(), "tags", "test", "test")
	require.NoError(t, err)

	taxons, err := db.GetTaxonomy(context.Background(), "tags")
	require.NoError(t, err)
	require.EqualValues(t, []string{"test"}, taxons)

	err = db.DeleteTaxonomy(context.Background(), "tags", "test")
	require.NoError(t, err)

	taxons, err = db.GetTaxonomy(context.Background(), "tags")
	require.NoError(t, err)
	require.EqualValues(t, []string{"test"}, taxons)

	err = db.DeleteTaxonomy(context.Background(), "tags", "test")
	require.NoError(t, err)

	taxons, err = db.GetTaxonomy(context.Background(), "tags")
	require.NoError(t, err)
	require.Len(t, taxons, 0)

	err = db.DeleteTaxonomy(context.Background(), "tags", "test")
	require.NoError(t, err)
}
