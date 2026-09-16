ALTER TABLE study_plans ADD COLUMN source_question_task_id BIGINT;
ALTER TABLE study_plans ADD COLUMN source_question_task_revision_id BIGINT;
ALTER TABLE study_plans ADD COLUMN task_claim_key VARCHAR(120);
CREATE UNIQUE INDEX idx_study_plans_task_claim ON study_plans(child_id, task_claim_key) WHERE task_claim_key IS NOT NULL;
CREATE INDEX idx_study_plans_task_revision ON study_plans(child_id, source_question_task_revision_id);
ALTER TABLE plan_items ALTER COLUMN question_id DROP NOT NULL;
ALTER TABLE plan_items ADD COLUMN question_version_id BIGINT;
ALTER TABLE plan_items ADD CONSTRAINT plan_items_question_source CHECK (question_id IS NOT NULL OR question_version_id IS NOT NULL);
CREATE TABLE question_attempt_receipts (
 id BIGSERIAL PRIMARY KEY, attempt_id BIGINT NOT NULL UNIQUE, child_id BIGINT NOT NULL, client_id VARCHAR(120) NOT NULL,
 plan_id BIGINT NOT NULL, plan_item_id BIGINT NOT NULL, question_version_id BIGINT NOT NULL,
 kp_id BIGINT NOT NULL, skill_code VARCHAR(80) NOT NULL, question_type VARCHAR(80) NOT NULL,
 selected_option_id VARCHAR(120) NOT NULL, display_index INT NOT NULL, original_index INT NOT NULL,
 is_correct BOOLEAN NOT NULL, cost_ms INT NOT NULL, response_json TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(child_id,client_id)
);
CREATE INDEX idx_question_receipts_plan ON question_attempt_receipts(child_id,plan_id,created_at);
-- A different request may resume the same unfinished plan. Retain every key.
CREATE TABLE study_plan_task_claims (
 child_id BIGINT NOT NULL, claim_key VARCHAR(120) NOT NULL,
 task_id BIGINT NOT NULL, revision_id BIGINT NOT NULL, plan_id BIGINT NOT NULL,
 PRIMARY KEY(child_id,claim_key)
);
