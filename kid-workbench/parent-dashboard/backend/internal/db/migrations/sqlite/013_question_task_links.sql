ALTER TABLE study_plans ADD COLUMN source_question_task_id INTEGER;
ALTER TABLE study_plans ADD COLUMN source_question_task_revision_id INTEGER;
ALTER TABLE study_plans ADD COLUMN task_claim_key TEXT;
CREATE UNIQUE INDEX idx_study_plans_task_claim ON study_plans(child_id,task_claim_key) WHERE task_claim_key IS NOT NULL;
CREATE INDEX idx_study_plans_task_revision ON study_plans(child_id,source_question_task_revision_id);
CREATE TABLE plan_items_with_versions (
 id INTEGER PRIMARY KEY AUTOINCREMENT, plan_id INTEGER NOT NULL REFERENCES study_plans(id) ON DELETE CASCADE,
 seq INTEGER NOT NULL, kp_id INTEGER NOT NULL REFERENCES knowledge_points(id), question_id INTEGER REFERENCES questions(id),
 bucket TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', tries INTEGER NOT NULL DEFAULT 0,
 cost_ms INTEGER NOT NULL DEFAULT 0, answered_at DATETIME, picks TEXT NOT NULL DEFAULT '', option_order TEXT NOT NULL DEFAULT '',
 question_stem TEXT NOT NULL DEFAULT '', question_options TEXT NOT NULL DEFAULT '', question_answer TEXT NOT NULL DEFAULT '',
 question_visual TEXT NOT NULL DEFAULT '', question_speech TEXT NOT NULL DEFAULT '', explanation TEXT NOT NULL DEFAULT '',
 content_snapshot_version INTEGER NOT NULL DEFAULT 0, question_snapshot TEXT NOT NULL DEFAULT '{}', question_version_id INTEGER,
 UNIQUE(plan_id,seq), CHECK(question_id IS NOT NULL OR question_version_id IS NOT NULL)
);
INSERT INTO plan_items_with_versions(id,plan_id,seq,kp_id,question_id,bucket,status,tries,cost_ms,answered_at,picks,option_order,question_stem,question_options,question_answer,question_visual,question_speech,explanation,content_snapshot_version,question_snapshot)
SELECT id,plan_id,seq,kp_id,question_id,bucket,status,tries,cost_ms,answered_at,picks,option_order,question_stem,question_options,question_answer,question_visual,question_speech,explanation,content_snapshot_version,question_snapshot FROM plan_items;
DROP TABLE plan_items;
ALTER TABLE plan_items_with_versions RENAME TO plan_items;
CREATE INDEX idx_plan_items_plan ON plan_items(plan_id,seq);
CREATE TABLE question_attempt_receipts (
 id INTEGER PRIMARY KEY AUTOINCREMENT, attempt_id INTEGER NOT NULL UNIQUE, child_id INTEGER NOT NULL, client_id TEXT NOT NULL,
 plan_id INTEGER NOT NULL, plan_item_id INTEGER NOT NULL, question_version_id INTEGER NOT NULL, kp_id INTEGER NOT NULL,
 skill_code TEXT NOT NULL, question_type TEXT NOT NULL, selected_option_id TEXT NOT NULL, display_index INTEGER NOT NULL,
 original_index INTEGER NOT NULL, is_correct BOOLEAN NOT NULL, cost_ms INTEGER NOT NULL, response_json TEXT NOT NULL,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, UNIQUE(child_id,client_id)
);
CREATE INDEX idx_question_receipts_plan ON question_attempt_receipts(child_id,plan_id,created_at);
-- A different request may resume the same unfinished plan. Retain every key.
CREATE TABLE study_plan_task_claims (
 child_id INTEGER NOT NULL, claim_key TEXT NOT NULL,
 task_id INTEGER NOT NULL, revision_id INTEGER NOT NULL, plan_id INTEGER NOT NULL,
 PRIMARY KEY(child_id,claim_key)
);
