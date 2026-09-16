package service_test

import (
	"context"
	"github.com/conchi/study-learning/learning"
	"github.com/conchi/study-learning/mastery"
	learningmodel "github.com/conchi/study-learning/model"
	"github.com/conchi/study-learning/pinyincatalog"
	"github.com/conchi/study-workbench/internal/config"
	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/seed"
	"github.com/conchi/study-workbench/internal/service"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"os"
	"strings"
	"testing"
	"time"
)

// Explicit dedicated database only. Never accepts the application's daily database.
func TestPinyinPostgresIntegration(t *testing.T) {
	name := os.Getenv("PINYIN_TEST_DB")
	if name == "" {
		t.Skip("set PINYIN_TEST_DB to a dedicated kid_pinyin_verify_* database")
	}
	require.True(t, strings.HasPrefix(name, "kid_pinyin_verify_"))
	c := config.Load()
	c.Pgsql.Dbname = name
	d, e := db.Open("postgres", c.Pgsql.Dsn())
	require.NoError(t, e)
	require.NoError(t, db.Migrate(d))
	require.NoError(t, db.Migrate(d))
	require.NoError(t, seed.Catalog(d))
	// This schema mirrors the content-owned catalog, not the production catalog.
	for _, sql := range []string{
		`CREATE TABLE IF NOT EXISTS pinyin_syllable_assets(id BIGINT PRIMARY KEY,initial_text TEXT,final_text TEXT,tone INT,syllable_text TEXT,speech_text TEXT,speech_url TEXT,difficulty INT,enabled BOOLEAN)`,
		`INSERT INTO pinyin_syllable_assets VALUES(90001,'b','ā',1,'bā','八','',1,true),(90002,'p','ā',1,'pā','趴','',1,true),(90003,'m','ā',1,'mā','妈','',1,true),(90004,'f','ā',1,'fā','发','',1,true) ON CONFLICT(id) DO NOTHING`,
	} {
		require.NoError(t, d.Exec(sql).Error)
	}
	// First request must include the just-synchronized syllables in the same total.
	dashboard := service.NewDashboardService(repo.New(d))
	overview, e := dashboard.Overview(1)
	require.NoError(t, e)
	require.Equal(t, overview.TotalKp, overview.Counts.NotStarted+overview.Counts.Learning+overview.Counts.Shaky+overview.Counts.Mastered+overview.Counts.ReviewDue)
	require.NoError(t, pinyincatalog.Sync(context.Background(), d))
	var link learningmodel.PinyinSyllableLink
	require.NoError(t, d.Where("asset_id=90001").First(&link).Error)
	cfg := mastery.DefaultConfig()
	cfg.BaseMasterStreak = 0
	learn := learning.NewService(cfg)
	require.NoError(t, d.Transaction(func(tx *gorm.DB) error {
		_, _, e := learn.ApplyOne(tx, 1, learning.AttemptInput{ClientID: "pg-blend", KpID: link.KpID, SkillCode: "blend", IsCorrect: true, At: time.Now()})
		return e
	}))
	m, e := dashboard.Matrix(1, "pinyin")
	require.NoError(t, e)
	require.Equal(t, 1, m.TypeProgress["blend"].Mastered)
	require.Equal(t, 4, m.TypeProgress["blend"].Total)
	before, e := dashboard.KpDetail(1, link.KpID)
	require.NoError(t, e)
	require.Equal(t, "mastered", before.Status)
	require.NotNil(t, before.MasteredAt)
	r, e := service.UpgradePinyin(d, true)
	require.NoError(t, e)
	require.GreaterOrEqual(t, r.AlreadyApplied, 1)
	after, e := dashboard.KpDetail(1, link.KpID)
	require.NoError(t, e)
	require.True(t, before.MasteredAt.Equal(*after.MasteredAt))
	require.NoError(t, d.Exec(`UPDATE pinyin_syllable_assets SET enabled=false WHERE id=90001`).Error)
	m, e = dashboard.Matrix(1, "pinyin")
	require.NoError(t, e)
	require.Equal(t, 3, m.TypeProgress["blend"].Total)
	require.Zero(t, m.TypeProgress["blend"].Mastered)
	require.NoError(t, d.Exec(`UPDATE pinyin_syllable_assets SET enabled=true WHERE id=90001`).Error)
	// Preparation for the separate formal-answer PostgreSQL integration test.
	var ids []int64
	require.NoError(t, d.Raw(`SELECT kp.id FROM knowledge_points kp JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='pinyin' AND m.code='shengmu' ORDER BY kp.id LIMIT 4`).Scan(&ids).Error)
	for i, id := range ids {
		letter := []string{"b", "p", "m", "f"}[i]
		require.NoError(t, d.Exec(`INSERT INTO pinyin_assets(kp_id,letter,module_code,module_name,module_order,kp_order,solo_text,word_text,glyph_image_url) VALUES(?,?,'shengmu','声母',1,?,'波','爸','/glyph.png') ON CONFLICT(kp_id) DO NOTHING`, id, letter, i).Error)
	}
}
