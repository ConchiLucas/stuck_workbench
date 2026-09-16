ALTER TABLE attempts ADD COLUMN IF NOT EXISTS plan_item_id BIGINT;
CREATE INDEX IF NOT EXISTS idx_attempts_plan_item ON attempts(plan_item_id);
