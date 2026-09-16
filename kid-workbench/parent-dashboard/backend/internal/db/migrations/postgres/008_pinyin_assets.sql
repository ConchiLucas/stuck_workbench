CREATE TABLE IF NOT EXISTS pinyin_assets (
  kp_id BIGINT PRIMARY KEY REFERENCES knowledge_points(id) ON DELETE CASCADE,
  letter VARCHAR(16) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  solo_text VARCHAR(16) NOT NULL DEFAULT '',
  word_text VARCHAR(16) NOT NULL DEFAULT '',
  solo_speech_url VARCHAR(512) NOT NULL DEFAULT '',
  word_speech_url VARCHAR(512) NOT NULL DEFAULT '',
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
