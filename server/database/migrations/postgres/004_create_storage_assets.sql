CREATE TABLE storage_assets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id BIGINT NULL REFERENCES business_plans(id) ON DELETE SET NULL,
    source VARCHAR(24) NOT NULL DEFAULT 'upload',
    name VARCHAR(255) NOT NULL,
    object_key VARCHAR(512) NOT NULL UNIQUE,
    mime_type VARCHAR(128) NOT NULL DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- statement-breakpoint
CREATE INDEX storage_assets_user_created_idx ON storage_assets(user_id, created_at DESC);
-- statement-breakpoint
CREATE INDEX storage_assets_source_idx ON storage_assets(source);
