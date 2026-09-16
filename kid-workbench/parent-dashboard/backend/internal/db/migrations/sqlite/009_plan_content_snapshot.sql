ALTER TABLE plan_items ADD COLUMN question_stem TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN question_options TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN question_answer TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN question_visual TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN question_speech TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN explanation TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN content_snapshot_version INTEGER NOT NULL DEFAULT 0;

UPDATE plan_items
SET question_stem = (SELECT stem FROM questions WHERE questions.id = plan_items.question_id),
    question_options = (SELECT options FROM questions WHERE questions.id = plan_items.question_id),
    question_answer = (SELECT answer FROM questions WHERE questions.id = plan_items.question_id),
    question_visual = (SELECT visual FROM questions WHERE questions.id = plan_items.question_id),
    question_speech = (SELECT speech FROM questions WHERE questions.id = plan_items.question_id),
    explanation = '',
    content_snapshot_version = 1
WHERE content_snapshot_version = 0
  AND EXISTS (SELECT 1 FROM questions WHERE questions.id = plan_items.question_id);
