-- 计划可按学科隔离（识字孩子端只看 literacy 计划）。
ALTER TABLE study_plans ADD COLUMN subject_code TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_study_plans_child_date_subject ON study_plans(child_id, plan_date, subject_code);
