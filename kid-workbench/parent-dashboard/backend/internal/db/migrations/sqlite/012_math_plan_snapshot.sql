ALTER TABLE plan_items ADD COLUMN question_snapshot TEXT NOT NULL DEFAULT '{}';
ALTER TABLE study_plans ADD COLUMN plan_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE study_plans ADD COLUMN module_code TEXT NOT NULL DEFAULT '';
ALTER TABLE study_plans ADD COLUMN stage_code TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_study_plans_math_scope
ON study_plans(child_id, subject_code, plan_kind, module_code, stage_code);
