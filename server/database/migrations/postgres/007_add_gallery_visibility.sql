-- Add visibility support for the public gallery feature.
-- Plans default to 'private'; users can opt-in to 'public' at analysis time.

ALTER TABLE business_plans ADD COLUMN visibility VARCHAR(16) NOT NULL DEFAULT 'private';

-- statement-breakpoint

ALTER TABLE business_plans ADD COLUMN admin_override BOOLEAN NOT NULL DEFAULT FALSE;

-- statement-breakpoint

-- Enable pg_trgm extension for fuzzy title matching (similarity detection).
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- statement-breakpoint

-- Index for gallery listing: only public + succeeded plans, ordered by newest.
CREATE INDEX idx_plans_gallery ON business_plans (visibility, status)
WHERE visibility = 'public';

-- statement-breakpoint

-- Trigram index on plan titles for fast similarity searches.
CREATE INDEX idx_plans_title_trgm ON business_plans USING gin (title gin_trgm_ops);

