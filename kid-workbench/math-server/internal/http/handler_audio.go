package http

import (
	"encoding/json"
	stdhttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func serveMathLearningAudio(db *gorm.DB, assets Assets) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, ok := positiveID(c, "childId", "invalid_child_id", "孩子编号无效")
		if !ok {
			return
		}
		kpID, ok := positiveID(c, "kpId", "invalid_kp_id", "知识点编号无效")
		if !ok {
			return
		}
		file := c.Param("file")
		if !strings.HasSuffix(file, ".mp3") {
			writeError(c, stdhttp.StatusNotFound, "audio_not_found", "音频范围不存在")
			return
		}
		code := strings.TrimSuffix(file, ".mp3")
		if code != "calc" && code != "story" && code != "find" && code != "name" {
			writeError(c, stdhttp.StatusNotFound, "audio_not_found", "音频范围不存在")
			return
		}
		var row struct{ MediaURL string }
		result := db.WithContext(c.Request.Context()).Raw(`
			SELECT q.media_url
			FROM questions q
			JOIN knowledge_points kp ON kp.id = q.kp_id
			JOIN modules m ON m.id = kp.module_id
			JOIN subjects s ON s.id = m.subject_id
			WHERE q.kp_id = ? AND q.code = ? AND s.code = 'math'
			  AND EXISTS(SELECT 1 FROM children c WHERE c.id = ?)`, kpID, code, childID).Scan(&row)
		if result.Error != nil {
			writeDatabaseUnavailable(c)
			return
		}
		if result.RowsAffected == 0 {
			writeError(c, stdhttp.StatusNotFound, "audio_not_found", "音频范围不存在")
			return
		}
		writeAudio(c, assets, row.MediaURL)
	}
}

func serveMathPlanAudio(db *gorm.DB, assets Assets) gin.HandlerFunc {
	return func(c *gin.Context) {
		childID, planID, ok := planIDs(c)
		if !ok {
			return
		}
		itemID, ok := positiveID(c, "itemId", "invalid_item_id", "题目编号无效")
		if !ok {
			return
		}
		var row struct{ QuestionSnapshot string }
		result := db.WithContext(c.Request.Context()).Raw(`
			SELECT pi.question_snapshot
			FROM plan_items pi
			JOIN study_plans p ON p.id = pi.plan_id
			WHERE pi.id = ? AND p.id = ? AND p.child_id = ?
			  AND p.subject_code = 'math' AND p.plan_kind <> ''`, itemID, planID, childID).Scan(&row)
		if result.Error != nil {
			writeDatabaseUnavailable(c)
			return
		}
		if result.RowsAffected == 0 {
			writeError(c, stdhttp.StatusNotFound, "audio_not_found", "音频范围不存在")
			return
		}
		var snapshot struct {
			AudioObjectKey string `json:"audioObjectKey"`
		}
		if json.Unmarshal([]byte(row.QuestionSnapshot), &snapshot) != nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "audio_unavailable", "音频暂时不可用，请重试")
			return
		}
		writeAudio(c, assets, snapshot.AudioObjectKey)
	}
}

func writeAudio(c *gin.Context, assets Assets, key string) {
	data, err := assets.QuestionAudio(c.Request.Context(), key)
	if err != nil {
		writeError(c, stdhttp.StatusServiceUnavailable, "audio_unavailable", "音频暂时不可用，请重试")
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.Data(stdhttp.StatusOK, "audio/mpeg", data)
}
