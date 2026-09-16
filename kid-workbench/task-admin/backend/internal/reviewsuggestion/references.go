package reviewsuggestion

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type reference struct {
	name, table, parent    string
	columns, parentColumns []string
}

// All identifiers are static migration metadata, never user input. PostgreSQL uses real
// RESTRICT foreign keys. SQLite existing-table upgrades use equivalent integrity triggers
// because ALTER TABLE ADD CONSTRAINT is unavailable; no learning business columns are changed.
func migrateReferences(g *gorm.DB) error {
	refs := []reference{
		{"rs_child", "review_suggestions", "children", []string{"child_id"}, []string{"id"}},
		{"rs_supersedes", "review_suggestions", "review_suggestions", []string{"supersedes_id"}, []string{"id"}},
		{"rs_target_parent", "review_suggestion_targets", "review_suggestions", []string{"suggestion_id"}, []string{"id"}},
		{"rs_target_kp", "review_suggestion_targets", "knowledge_points", []string{"kp_id"}, []string{"id"}},
		{"rs_evidence_target", "review_suggestion_evidence", "review_suggestion_targets", []string{"target_id"}, []string{"id"}},
		{"rs_evidence_attempt", "review_suggestion_evidence", "attempts", []string{"attempt_id"}, []string{"id"}},
		{"rs_evidence_receipt", "review_suggestion_evidence", "question_attempt_receipts", []string{"source_receipt_id"}, []string{"id"}},
		{"rs_evidence_version", "review_suggestion_evidence", "question_versions", []string{"source_question_version_id"}, []string{"id"}},
		{"rs_evidence_plan", "review_suggestion_evidence", "study_plans", []string{"source_plan_id"}, []string{"id"}},
		{"rs_evidence_item", "review_suggestion_evidence", "plan_items", []string{"source_plan_item_id"}, []string{"id"}},
		{"rs_evidence_pinyin", "review_suggestion_evidence", "pinyin_quiz_instances", []string{"source_pinyin_instance_id"}, []string{"id"}},
		{"rs_evidence_science", "review_suggestion_evidence", "science_attempt_receipts", []string{"source_child_id", "source_client_id"}, []string{"child_id", "client_id"}},
		{"rs_run_parent", "review_suggestion_runs", "review_suggestions", []string{"suggestion_id"}, []string{"id"}},
		{"rs_partition_parent", "review_suggestion_partitions", "review_suggestions", []string{"suggestion_id"}, []string{"id"}},
		{"rs_partition_run", "review_suggestion_partitions", "review_suggestion_runs", []string{"run_id"}, []string{"id"}},
		{"rs_link_parent", "review_suggestion_tasks", "review_suggestions", []string{"suggestion_id"}, []string{"id"}},
		{"rs_link_partition", "review_suggestion_tasks", "review_suggestion_partitions", []string{"partition_id"}, []string{"id"}},
		{"rs_link_task", "review_suggestion_tasks", "question_tasks", []string{"task_id"}, []string{"id"}},
		{"rs_link_revision", "review_suggestion_tasks", "question_task_revisions", []string{"generated_revision_id"}, []string{"id"}},
		{"rs_command_parent", "review_suggestion_commands", "review_suggestions", []string{"suggestion_id"}, []string{"id"}},
		{"rs_command_run", "review_suggestion_commands", "review_suggestion_runs", []string{"run_id"}, []string{"id"}},
	}
	// Earlier builds used zero for commands that do not create a run.
	if e := g.Exec("UPDATE review_suggestion_commands SET run_id=NULL WHERE run_id=0").Error; e != nil {
		return e
	}
	if e := backfillReferences(g); e != nil {
		return e
	}
	for _, r := range refs {
		if !g.Migrator().HasTable(r.parent) {
			continue
		}
		if g.Dialector.Name() == "postgres" {
			var n int64
			if e := g.Raw("SELECT COUNT(*) FROM pg_constraint WHERE conname=? AND conrelid=?::regclass", r.name, r.table).Scan(&n).Error; e != nil {
				return e
			}
			if n == 0 {
				sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON UPDATE RESTRICT ON DELETE RESTRICT", r.table, r.name, strings.Join(r.columns, ","), r.parent, strings.Join(r.parentColumns, ","))
				if e := g.Exec(sql).Error; e != nil {
					return e
				}
			}
			continue
		}
		if g.Dialector.Name() != "sqlite" {
			continue
		}
		present, matches, referenced, changed := []string{}, []string{}, []string{}, []string{}
		for i, c := range r.columns {
			pc := r.parentColumns[i]
			present = append(present, "NEW."+c+" IS NOT NULL")
			matches = append(matches, pc+"=NEW."+c)
			referenced = append(referenced, c+"=OLD."+pc)
			changed = append(changed, "NEW."+pc+" IS NOT OLD."+pc)
		}
		for _, event := range []string{"INSERT", "UPDATE"} {
			sql := fmt.Sprintf("CREATE TRIGGER IF NOT EXISTS %s_%s BEFORE %s ON %s WHEN %s AND NOT EXISTS (SELECT 1 FROM %s WHERE %s) BEGIN SELECT RAISE(ABORT,'review suggestion foreign key restriction'); END", r.name, strings.ToLower(event), event, r.table, strings.Join(present, " AND "), r.parent, strings.Join(matches, " AND "))
			if e := g.Exec(sql).Error; e != nil {
				return e
			}
		}
		for _, event := range []string{"DELETE", "UPDATE"} {
			when := ""
			if event == "UPDATE" {
				when = "(" + strings.Join(changed, " OR ") + ") AND "
			}
			sql := fmt.Sprintf("CREATE TRIGGER IF NOT EXISTS %s_parent_%s BEFORE %s ON %s WHEN %sEXISTS (SELECT 1 FROM %s WHERE %s) BEGIN SELECT RAISE(ABORT,'review suggestion foreign key restriction'); END", r.name, strings.ToLower(event), event, r.parent, when, r.table, strings.Join(referenced, " AND "))
			if e := g.Exec(sql).Error; e != nil {
				return e
			}
		}
	}
	return nil
}

// Populate newly indexed references from previously server-verified immutable snapshots.
func backfillReferences(g *gorm.DB) error {
	var rows []Evidence
	if e := g.Where("source_receipt_id IS NULL AND source_question_version_id IS NULL AND source_plan_id IS NULL AND source_pinyin_instance_id IS NULL AND source_child_id IS NULL").Find(&rows).Error; e != nil {
		return e
	}
	for _, row := range rows {
		var source Source
		var summary verified
		if e := json.Unmarshal([]byte(row.SourceJSON), &source); e != nil {
			return fmt.Errorf("review evidence %d has invalid source JSON: %w", row.ID, e)
		}
		if e := json.Unmarshal([]byte(row.EvidenceSummaryJSON), &summary); e != nil {
			return fmt.Errorf("review evidence %d has invalid summary JSON: %w", row.ID, e)
		}
		fields := map[string]any{}
		switch source.Kind {
		case "literacy_version":
			fields["source_receipt_id"] = source.ReceiptID
			fields["source_question_version_id"] = source.QuestionVersionID
			if summary.PlanID > 0 && summary.ItemID > 0 {
				fields["source_plan_id"] = summary.PlanID
				fields["source_plan_item_id"] = summary.ItemID
			}
		case "pinyin_instance":
			fields["source_pinyin_instance_id"] = source.InstanceID
		case "science_plan":
			var child int64
			if e := g.Table("review_suggestion_targets t").Select("s.child_id").Joins("JOIN review_suggestions s ON s.id=t.suggestion_id").Where("t.id=?", row.TargetID).Scan(&child).Error; e != nil {
				return e
			}
			fields["source_child_id"] = child
			fields["source_client_id"] = source.ClientID
			fields["source_plan_id"] = source.PlanID
			fields["source_plan_item_id"] = source.ItemID
		case "legacy_attempt":
			continue
		default:
			return fmt.Errorf("review evidence %d has unknown source kind", row.ID)
		}
		if e := g.Model(&Evidence{}).Where("id=?", row.ID).Updates(fields).Error; e != nil {
			return e
		}
	}
	return nil
}
