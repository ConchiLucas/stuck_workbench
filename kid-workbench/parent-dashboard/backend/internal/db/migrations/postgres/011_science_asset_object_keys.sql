CREATE TABLE IF NOT EXISTS science_assets (
  kp_id BIGINT PRIMARY KEY,
  title VARCHAR(64) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  needs_sense_image BOOLEAN NOT NULL DEFAULT FALSE,
  needs_sense_image_override BOOLEAN,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  sense_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  explanation TEXT NOT NULL DEFAULT '',
  fun_fact TEXT NOT NULL DEFAULT '',
  review_status VARCHAR(16) NOT NULL DEFAULT 'draft',
  reviewed_at TIMESTAMPTZ,
  content_version INT NOT NULL DEFAULT 1,
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS glyph_object_key TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS sense_object_key TEXT NOT NULL DEFAULT '';
ALTER TABLE science_assets ADD COLUMN IF NOT EXISTS speech_object_key TEXT NOT NULL DEFAULT '';
