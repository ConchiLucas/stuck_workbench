ALTER TABLE plan_items
ADD COLUMN IF NOT EXISTS question_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE study_plans
ADD COLUMN IF NOT EXISTS plan_kind VARCHAR(16) NOT NULL DEFAULT '';

ALTER TABLE study_plans
ADD COLUMN IF NOT EXISTS module_code VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE study_plans
ADD COLUMN IF NOT EXISTS stage_code VARCHAR(32) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_study_plans_math_scope
ON study_plans(child_id, subject_code, plan_kind, module_code, stage_code);
