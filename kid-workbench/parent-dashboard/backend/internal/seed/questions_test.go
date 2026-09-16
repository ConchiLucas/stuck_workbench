package seed_test

import (
	"encoding/json"
	"testing"

	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/seed"
)

func TestQuestionsCoverEnabledSubjectsOnly(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}

	stats, err := seed.Questions(gdb)
	if err != nil {
		t.Fatal(err)
	}

	// 全库可出题知识点：古诗由 EnsureMaterials 写题，quiz.Generate 不计。
	if stats.Kps != 1190 {
		t.Errorf("可出题知识点 = %d，期望 1190", stats.Kps)
	}
	if stats.Questions < 1100 {
		t.Errorf("题目数 = %d，太少了", stats.Questions)
	}
	for _, code := range []string{"math", "pinyin", "literacy", "english", "science", "poem", "logic", "chengyu", "phrase"} {
		if stats.BySubject[code] == 0 {
			t.Errorf("%s 没有生成任何题目", code)
		}
	}

	// 游戏科仍未开启，一道题都不该有。
	var leaked int64
	gdb.Raw(`SELECT COUNT(1) FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.quiz_enabled = ?`, false).Scan(&leaked)
	if leaked != 0 {
		t.Errorf("未开启出题的学科出现了 %d 道题", leaked)
	}
}

func TestQuestionsAreIdempotent(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}

	first, err := seed.Questions(gdb)
	if err != nil {
		t.Fatal(err)
	}
	var afterFirst int64
	gdb.Raw(`SELECT COUNT(1) FROM questions`).Scan(&afterFirst)

	if _, err := seed.Questions(gdb); err != nil {
		t.Fatal(err)
	}
	var afterSecond int64
	gdb.Raw(`SELECT COUNT(1) FROM questions`).Scan(&afterSecond)

	if afterFirst != afterSecond {
		t.Errorf("重复灌库产生了新记录：%d → %d", afterFirst, afterSecond)
	}
	if afterFirst != int64(first.Questions) {
		t.Errorf("入库数 %d 与统计 %d 不一致", afterFirst, first.Questions)
	}
}

func TestScienceReseedRemovesLegacyFactVariants(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}

	var kpID int64
	if err := gdb.Raw(`SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'science' LIMIT 1`).Scan(&kpID).Error; err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"fact1", "fact2"} {
		if err := gdb.Exec(`INSERT INTO questions
			(kp_id, code, type, stem, options, answer, visual, speech, difficulty)
			VALUES (?, ?, 'choice', 'legacy', '[]', '{}', '{}', '{}', 1)`, kpID, code).Error; err != nil {
			t.Fatal(err)
		}
	}

	if _, err := seed.Questions(gdb); err != nil {
		t.Fatal(err)
	}
	var legacyCount, recognizeCount int64
	gdb.Raw(`SELECT COUNT(*) FROM questions WHERE kp_id = ? AND code IN ('fact1','fact2')`, kpID).Scan(&legacyCount)
	gdb.Raw(`SELECT COUNT(*) FROM questions WHERE kp_id = ? AND code = 'recognize'`, kpID).Scan(&recognizeCount)
	if legacyCount != 0 || recognizeCount != 1 {
		t.Fatalf("legacy=%d recognize=%d，期望 0/1", legacyCount, recognizeCount)
	}
}

func TestReseedKeepsLegacyQuestionReferencedByPlan(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Questions(gdb); err != nil {
		t.Fatal(err)
	}

	var kpID int64
	if err := gdb.Raw(`SELECT kp.id FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'science' LIMIT 1`).Scan(&kpID).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`INSERT INTO questions
		(kp_id, code, type, stem, options, answer, visual, speech, difficulty)
		VALUES (?, 'legacy_fact', 'choice', 'legacy', '[]', '{}', '{}', '{}', 1)`, kpID).Error; err != nil {
		t.Fatal(err)
	}
	var questionID int64
	if err := gdb.Raw(`SELECT id FROM questions WHERE kp_id = ? AND code = 'legacy_fact'`, kpID).Scan(&questionID).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`INSERT INTO study_plans(id, child_id, plan_date, seq_no, subject_code, status, target_count, created_at)
		VALUES (1, 1, '2026-09-01', 1, 'science', 'completed', 1, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`INSERT INTO plan_items(plan_id, seq, kp_id, question_id, bucket, status)
		VALUES (1, 1, ?, ?, 'new', 'completed')`, kpID, questionID).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := seed.Questions(gdb); err != nil {
		t.Fatal(err)
	}
	var count int64
	gdb.Raw(`SELECT COUNT(*) FROM questions WHERE id = ?`, questionID).Scan(&count)
	if count != 1 {
		t.Fatalf("历史题单引用的题被删除，count=%d", count)
	}
}

func TestStoredQuestionsAreAnswerable(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Questions(gdb); err != nil {
		t.Fatal(err)
	}

	type row struct {
		ID      int64
		Code    string
		Stem    string
		Options string
		Answer  string
	}
	var rows []row
	if err := gdb.Raw(`SELECT id, code, stem, options, answer FROM questions`).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		if r.Code == "pattern" || r.Code == "classify" || r.Code == "order" || r.Code == "shape_reason" || r.Code == "diff" || r.Code == "compare" || r.Code == "title" || r.Code == "fill" || r.Code == "couplet" || r.Code == "recite" {
			if r.Stem == "" {
				t.Fatalf("题 %d 题干为空", r.ID)
			}
			continue
		}
		var opts []map[string]any
		if err := json.Unmarshal([]byte(r.Options), &opts); err != nil {
			t.Fatalf("题 %d 选项不是合法 JSON: %v", r.ID, err)
		}
		var ans struct{ Index int }
		if err := json.Unmarshal([]byte(r.Answer), &ans); err != nil {
			t.Fatalf("题 %d 答案不是合法 JSON: %v", r.ID, err)
		}
		if r.Code == "write_char" {
			if len(opts) != 0 {
				t.Fatalf("写字题 %d 不该有选项", r.ID)
			}
			if ans.Index != 0 {
				t.Fatalf("写字题 %d 正确项下标 %d", r.ID, ans.Index)
			}
		} else {
			if len(opts) != 4 {
				t.Fatalf("题 %d 有 %d 个选项", r.ID, len(opts))
			}
			if ans.Index < 0 || ans.Index >= len(opts) {
				t.Fatalf("题 %d 正确项下标 %d 越界", r.ID, ans.Index)
			}
		}
		if r.Stem == "" {
			t.Fatalf("题 %d 题干为空", r.ID)
		}
	}
}
