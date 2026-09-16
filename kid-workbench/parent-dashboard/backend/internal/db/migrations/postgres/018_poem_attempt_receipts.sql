CREATE TABLE IF NOT EXISTS poem_attempt_receipts (
  child_id BIGINT NOT NULL,
  client_id VARCHAR(120) NOT NULL,
  kp_id BIGINT NOT NULL,
  question_id BIGINT NOT NULL,
  plan_id BIGINT NOT NULL,
  item_id BIGINT NOT NULL,
  result_json TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (child_id, client_id)
);
