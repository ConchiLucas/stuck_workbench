CREATE TABLE question_attempt_receipts_v2 (
 id INTEGER PRIMARY KEY AUTOINCREMENT, attempt_id INTEGER NOT NULL UNIQUE, child_id INTEGER NOT NULL, client_id TEXT NOT NULL,
 plan_id INTEGER NOT NULL, plan_item_id INTEGER NOT NULL, question_version_id INTEGER NOT NULL, kp_id INTEGER NOT NULL,
 skill_code TEXT NOT NULL, question_type TEXT NOT NULL, selected_option_id TEXT, display_index INTEGER,
 original_index INTEGER, is_correct BOOLEAN NOT NULL, cost_ms INTEGER NOT NULL, response_json TEXT NOT NULL,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, response_kind TEXT NOT NULL DEFAULT 'choice',
 answer_payload_json TEXT NOT NULL DEFAULT '{}',evaluation_json TEXT NOT NULL DEFAULT '{}',evaluator_version TEXT NOT NULL DEFAULT 'choice-v1',UNIQUE(child_id,client_id)
);
INSERT INTO question_attempt_receipts_v2(id,attempt_id,child_id,client_id,plan_id,plan_item_id,question_version_id,kp_id,skill_code,question_type,selected_option_id,display_index,original_index,is_correct,cost_ms,response_json,created_at)
SELECT id,attempt_id,child_id,client_id,plan_id,plan_item_id,question_version_id,kp_id,skill_code,question_type,selected_option_id,display_index,original_index,is_correct,cost_ms,response_json,created_at FROM question_attempt_receipts;
DROP TABLE question_attempt_receipts;
ALTER TABLE question_attempt_receipts_v2 RENAME TO question_attempt_receipts;
CREATE INDEX idx_question_receipts_plan ON question_attempt_receipts(child_id,plan_id,created_at);
CREATE TABLE literacy_writing_template_cache (revision_id TEXT PRIMARY KEY,sha256 TEXT NOT NULL,template_json TEXT NOT NULL);
-- Older deployed copies of migration 013 predate the retained-claim-key table.
CREATE TABLE IF NOT EXISTS study_plan_task_claims (
 child_id INTEGER NOT NULL, claim_key TEXT NOT NULL,
 task_id INTEGER NOT NULL, revision_id INTEGER NOT NULL, plan_id INTEGER NOT NULL,
 PRIMARY KEY(child_id,claim_key)
);
