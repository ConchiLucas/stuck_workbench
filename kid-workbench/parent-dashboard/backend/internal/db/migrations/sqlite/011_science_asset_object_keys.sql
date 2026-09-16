CREATE TABLE IF NOT EXISTS science_assets (
  kp_id INTEGER PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  needs_sense_image INTEGER NOT NULL DEFAULT 0,
  needs_sense_image_override INTEGER,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  explanation TEXT NOT NULL DEFAULT '',
  fun_fact TEXT NOT NULL DEFAULT '',
  review_status TEXT NOT NULL DEFAULT 'draft',
  reviewed_at DATETIME,
  content_version INTEGER NOT NULL DEFAULT 1,
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE science_assets ADD COLUMN glyph_object_key TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN sense_object_key TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN speech_object_key TEXT NOT NULL DEFAULT '';
