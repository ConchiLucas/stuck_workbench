package pinyincatalog_test

import (
	"context"
	"github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestSyncPreservesIdentityAndDisablesMissingContent(t *testing.T) {
	d, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, e)
	require.NoError(t, d.AutoMigrate(&model.KnowledgePoint{}, &model.PinyinSyllableLink{}))
	for _, q := range []string{
		`CREATE TABLE subjects(id INTEGER PRIMARY KEY,code TEXT UNIQUE)`,
		`INSERT INTO subjects VALUES(1,'pinyin')`,
		`CREATE TABLE modules(id INTEGER PRIMARY KEY,subject_id INTEGER,code TEXT,name TEXT,order_no INTEGER,UNIQUE(subject_id,code))`,
		`CREATE UNIQUE INDEX kp_module_code ON knowledge_points(module_id,code)`,
		`CREATE TABLE pinyin_syllable_assets(id INTEGER PRIMARY KEY,initial_text TEXT,final_text TEXT,tone INT,syllable_text TEXT,difficulty INT,enabled BOOLEAN)`,
		`INSERT INTO pinyin_syllable_assets VALUES(30,'b','ā',1,'bā',1,true)`,
	} {
		require.NoError(t, d.Exec(q).Error)
	}
	require.NoError(t, pinyincatalog.Sync(context.Background(), d))
	var one model.PinyinSyllableLink
	require.NoError(t, d.First(&one).Error)
	require.NotEqual(t, int64(30), one.KpID)
	require.NoError(t, pinyincatalog.Sync(context.Background(), d))
	var n int64
	d.Model(&model.KnowledgePoint{}).Count(&n)
	require.Equal(t, int64(1), n)
	require.NoError(t, d.Exec(`UPDATE pinyin_syllable_assets SET tone=2`).Error)
	require.ErrorIs(t, pinyincatalog.Sync(context.Background(), d), pinyincatalog.ErrIdentityConflict)
	require.NoError(t, d.Exec(`UPDATE pinyin_syllable_assets SET tone=1,enabled=false`).Error)
	require.NoError(t, pinyincatalog.Sync(context.Background(), d))
	var after model.PinyinSyllableLink
	d.First(&after)
	require.False(t, after.Enabled)
	require.Equal(t, one.KpID, after.KpID)
}
func TestMissingCatalogIsNotEmptySuccess(t *testing.T) {
	d, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, e)
	require.ErrorIs(t, pinyincatalog.Sync(context.Background(), d), pinyincatalog.ErrUnavailable)
}
