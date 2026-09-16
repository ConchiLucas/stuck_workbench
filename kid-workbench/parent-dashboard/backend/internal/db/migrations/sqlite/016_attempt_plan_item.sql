ALTER TABLE attempts ADD COLUMN plan_item_id INTEGER;
CREATE INDEX IF NOT EXISTS idx_attempts_plan_item ON attempts(plan_item_id);
