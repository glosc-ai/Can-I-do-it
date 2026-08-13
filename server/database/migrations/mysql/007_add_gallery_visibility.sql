-- Add visibility support for the public gallery feature.
-- Plans default to 'private'; users can opt-in to 'public' at analysis time.

ALTER TABLE business_plans ADD COLUMN visibility VARCHAR(16) NOT NULL DEFAULT 'private';

-- statement-breakpoint

ALTER TABLE business_plans ADD COLUMN admin_override TINYINT(1) NOT NULL DEFAULT 0;

-- statement-breakpoint

-- Index for gallery listing queries.
CREATE INDEX idx_plans_gallery ON business_plans (visibility, status);

-- statement-breakpoint

-- Fulltext index on plan titles for similarity/search queries (MySQL 5.7+).
CREATE FULLTEXT INDEX idx_plans_title_ft ON business_plans (title);

