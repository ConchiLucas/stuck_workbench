package db_test

import (
	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMaterialMigrationUpgradesAndPreservesRows(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`CREATE TABLE material_revisions (revision_id TEXT PRIMARY KEY, subject_code TEXT NOT NULL, kp_id BIGINT NOT NULL, content TEXT NOT NULL, media TEXT NOT NULL, created_at TIMESTAMP NOT NULL)`).Error)
	require.NoError(t, gdb.Exec(`INSERT INTO material_revisions VALUES ('old','literacy',1,'{}','{}',CURRENT_TIMESTAMP)`).Error)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, db.Migrate(gdb))
	require.True(t, gdb.Migrator().HasColumn("material_revisions", "source_revision"))
	var source string
	require.NoError(t, gdb.Raw(`SELECT source_revision FROM material_revisions WHERE revision_id='old'`).Scan(&source).Error)
	require.Empty(t, source)
	require.NoError(t, gdb.Create(&literacy.Asset{KpID: 1, CharText: "山"}).Error)
	var asset literacy.Asset
	require.NoError(t, gdb.First(&asset, "kp_id=1").Error)
	require.Empty(t, asset.MaterialEpoch)
	require.False(t, asset.MaterialPending)
}
