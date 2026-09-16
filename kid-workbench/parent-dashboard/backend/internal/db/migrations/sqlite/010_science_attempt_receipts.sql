CREATE TABLE IF NOT EXISTS science_attempt_receipts (
  child_id INTEGER NOT NULL,
  client_id TEXT NOT NULL,
  kp_id INTEGER NOT NULL,
  question_id INTEGER NOT NULL,
  plan_id INTEGER NOT NULL,
  item_id INTEGER NOT NULL,
  result_json TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (child_id, client_id)
);
