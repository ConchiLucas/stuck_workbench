CREATE TABLE pinyin_syllable_links (
 asset_id INTEGER PRIMARY KEY,
 kp_id INTEGER NOT NULL UNIQUE REFERENCES knowledge_points(id),
 initial_text TEXT NOT NULL, final_text TEXT NOT NULL, tone INT NOT NULL,
 syllable_text TEXT NOT NULL, enabled BOOLEAN NOT NULL, synced_at DATETIME NOT NULL,
 UNIQUE(initial_text, final_text, tone)
);
CREATE TABLE pinyin_quiz_instances (
 id TEXT PRIMARY KEY, child_id INTEGER NOT NULL REFERENCES children(id),
 kp_id INTEGER NOT NULL REFERENCES knowledge_points(id), skill_code TEXT NOT NULL,
 snapshot_version INT NOT NULL, public_snapshot TEXT NOT NULL, answer_option_id TEXT NOT NULL,
 created_at DATETIME NOT NULL, expires_at DATETIME NOT NULL
);
CREATE INDEX idx_pinyin_instances_child ON pinyin_quiz_instances(child_id, created_at);
CREATE TABLE pinyin_answer_receipts (
 child_id INTEGER NOT NULL REFERENCES children(id), client_id TEXT NOT NULL,
 instance_id TEXT NOT NULL UNIQUE REFERENCES pinyin_quiz_instances(id),
 attempt_id INTEGER NOT NULL UNIQUE REFERENCES attempts(id), skill_code TEXT NOT NULL,
 selected_option_id TEXT NOT NULL, response_snapshot TEXT NOT NULL, created_at DATETIME NOT NULL,
 PRIMARY KEY(child_id, client_id)
);
CREATE TABLE pinyin_mastery_milestones (
 child_id INTEGER NOT NULL REFERENCES children(id), kp_id INTEGER NOT NULL REFERENCES knowledge_points(id),
 rule_version INT NOT NULL, first_completed_at DATETIME NOT NULL,
 PRIMARY KEY(child_id, kp_id, rule_version)
);
CREATE TABLE pinyin_upgrade_audits (
 child_id INTEGER NOT NULL REFERENCES children(id), kp_id INTEGER NOT NULL REFERENCES knowledge_points(id),
 rule_version INT NOT NULL, before_snapshot TEXT NOT NULL, had_reward BOOLEAN NOT NULL,
 applied_at DATETIME NOT NULL, PRIMARY KEY(child_id, kp_id, rule_version)
);
