-- Add file size and analysis duration tracking to documents.
ALTER TABLE documents
  ADD COLUMN IF NOT EXISTS file_size_bytes      BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS analysis_duration_ms BIGINT;
