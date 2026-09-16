ALTER TABLE question_attempt_receipts ALTER COLUMN selected_option_id DROP NOT NULL;
ALTER TABLE question_attempt_receipts ALTER COLUMN display_index DROP NOT NULL;
ALTER TABLE question_attempt_receipts ALTER COLUMN original_index DROP NOT NULL;
ALTER TABLE question_attempt_receipts ADD COLUMN response_kind TEXT NOT NULL DEFAULT 'choice';
ALTER TABLE question_attempt_receipts ADD COLUMN answer_payload_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE question_attempt_receipts ADD COLUMN evaluation_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE question_attempt_receipts ADD COLUMN evaluator_version TEXT NOT NULL DEFAULT 'choice-v1';
CREATE TABLE literacy_writing_template_cache (revision_id TEXT PRIMARY KEY,sha256 TEXT NOT NULL,template_json TEXT NOT NULL);
-- Older deployed copies of migration 013 predate the retained-claim-key table.
CREATE TABLE IF NOT EXISTS study_plan_task_claims (
 child_id BIGINT NOT NULL, claim_key VARCHAR(120) NOT NULL,
 task_id BIGINT NOT NULL, revision_id BIGINT NOT NULL, plan_id BIGINT NOT NULL,
 PRIMARY KEY(child_id,claim_key)
);
