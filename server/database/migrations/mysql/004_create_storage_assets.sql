CREATE TABLE storage_assets (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    plan_id BIGINT NULL,
    source VARCHAR(24) NOT NULL DEFAULT 'upload',
    name VARCHAR(255) NOT NULL,
    object_key VARCHAR(512) NOT NULL UNIQUE,
    mime_type VARCHAR(128) NOT NULL DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT assets_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT assets_plan_fk FOREIGN KEY (plan_id) REFERENCES business_plans(id) ON DELETE SET NULL
);
-- statement-breakpoint
CREATE INDEX storage_assets_user_created_idx ON storage_assets(user_id, created_at DESC);
-- statement-breakpoint
CREATE INDEX storage_assets_source_idx ON storage_assets(source);
