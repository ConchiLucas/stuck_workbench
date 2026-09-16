package seed

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/quiz"
	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-learning/poemcontent"
)

// QuestionStats 是灌库结果，用来在 CLI 里核对覆盖情况。
type QuestionStats struct {
	Kps       int            // 可出题的知识点数
	Questions int            // 写入的题目数
	Skipped   int            // 生成器不支持、跳过的知识点数
	BySubject map[string]int // 各学科题目数
}

// Questions 为所有 quiz_enabled 学科的知识点生成题目并写入 questions 表。
// 幂等：按 (kp_id, code) upsert，重复执行只会覆盖内容、不会产生重复题。
func Questions(gdb *gorm.DB) (QuestionStats, error) {
	stats := QuestionStats{BySubject: map[string]int{}}

	type row struct {
		KpID        int64
		Title       string
		Payload     string
		Difficulty  int
		ModuleID    int64
		ModuleCode  string
		SubjectCode string
	}
	var rows []row
	err := gdb.Raw(`
		SELECT kp.id AS kp_id, kp.title, kp.payload, kp.difficulty,
		       m.id AS module_id, m.code AS module_code, s.code AS subject_code
		FROM knowledge_points kp
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.quiz_enabled = ?
		ORDER BY s.order_no, m.order_no, kp.order_no`, true).Scan(&rows).Error
	if err != nil {
		return stats, err
	}

	// 干扰项来自同模块的兄弟节点，先按模块聚一遍标题。
	siblings := map[int64][]string{}
	siblingIDs := map[int64]map[string]int64{}
	siblingZh := map[int64]map[string]int64{}
	for _, r := range rows {
		siblings[r.ModuleID] = append(siblings[r.ModuleID], r.Title)
		if siblingIDs[r.ModuleID] == nil {
			siblingIDs[r.ModuleID] = map[string]int64{}
		}
		siblingIDs[r.ModuleID][r.Title] = r.KpID
		if r.SubjectCode == "phrase" {
			if siblingZh[r.ModuleID] == nil {
				siblingZh[r.ModuleID] = map[string]int64{}
			}
			var payload struct {
				Zh string `json:"zh"`
			}
			_ = json.Unmarshal([]byte(r.Payload), &payload)
			if strings.TrimSpace(payload.Zh) != "" {
				siblingZh[r.ModuleID][strings.TrimSpace(payload.Zh)] = r.KpID
			}
		}
	}

	err = gdb.Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			specs := quiz.Generate(quiz.Kp{
				ID: r.KpID, Title: r.Title, Payload: r.Payload, Difficulty: r.Difficulty,
				SubjectCode: r.SubjectCode, ModuleCode: r.ModuleCode,
				Siblings: siblings[r.ModuleID], SiblingIDs: siblingIDs[r.ModuleID], SiblingZhIDs: siblingZh[r.ModuleID],
			})
			if len(specs) == 0 {
				stats.Skipped++
				continue
			}
			stats.Kps++

			for _, sp := range specs {
				q := model.Question{
					KpID: r.KpID, Code: sp.Code, Type: questionType(sp.Code), Stem: sp.Stem,
					Options: sp.OptionsJSON(), Answer: sp.AnswerJSON(),
					Visual: sp.VisualJSON(), Speech: sp.SpeechJSON(),
					Difficulty: sp.Difficulty,
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "kp_id"}, {Name: "code"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"type", "stem", "options", "answer", "visual", "speech", "difficulty",
					}),
				}).Create(&q).Error; err != nil {
					return err
				}
				stats.Questions++
				stats.BySubject[r.SubjectCode]++
			}
			if r.SubjectCode == "science" {
				if err := tx.Where("kp_id = ? AND code IN ?", r.KpID, []string{"fact1", "fact2"}).
					Where("NOT EXISTS (SELECT 1 FROM plan_items pi WHERE pi.question_id = questions.id)").
					Delete(&model.Question{}).Error; err != nil {
					return err
				}
			}
		}
		if err := poemcontent.EnsureMaterials(tx); err != nil {
			return err
		}
		if err := logiccontent.EnsureMaterials(tx); err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		var poemCount, logicCount int64
		_ = gdb.Raw(`SELECT COUNT(*) FROM questions q JOIN knowledge_points kp ON kp.id=q.kp_id JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='poem'`).Scan(&poemCount).Error
		stats.BySubject["poem"] = int(poemCount)
		if poemCount > 0 {
			stats.Questions += int(poemCount)
		}
		_ = gdb.Raw(`SELECT COUNT(*) FROM questions q JOIN knowledge_points kp ON kp.id=q.kp_id JOIN modules m ON m.id=kp.module_id JOIN subjects s ON s.id=m.subject_id WHERE s.code='logic' AND q.code IN ('pattern','classify','order','shape_reason','diff','compare')`).Scan(&logicCount).Error
		if logicCount > 0 {
			stats.BySubject["logic"] += int(logicCount)
			stats.Questions += int(logicCount)
		}
	}
	return stats, err
}

func questionType(code string) string {
	if code == "write_char" {
		return "write"
	}
	return "choice"
}
