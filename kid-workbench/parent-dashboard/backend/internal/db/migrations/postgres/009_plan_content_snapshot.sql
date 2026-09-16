ALTER TABLE plan_items ADD COLUMN IF NOT EXISTS question_stem TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN IF NOT EXISTS question_options TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN IF NOT EXISTS question_answer TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN IF NOT EXISTS question_visual TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN IF NOT EXISTS question_speech TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN IF NOT EXISTS explanation TEXT NOT NULL DEFAULT '';
ALTER TABLE plan_items ADD COLUMN IF NOT EXISTS content_snapshot_version INT NOT NULL DEFAULT 0;

UPDATE plan_items pi
SET question_stem = q.stem,
    question_options = q.options,
    question_answer = q.answer,
    question_visual = q.visual,
    question_speech = q.speech,
    explanation = '',
    content_snapshot_version = 1
FROM questions q
WHERE pi.question_id = q.id AND pi.content_snapshot_version = 0;
