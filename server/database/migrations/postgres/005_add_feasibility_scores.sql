CREATE TABLE analysis_dimension_scores (
    id BIGSERIAL PRIMARY KEY,
    analysis_job_id BIGINT NOT NULL REFERENCES analysis_jobs(id) ON DELETE CASCADE,
    dimension_key VARCHAR(64) NOT NULL,
    dimension_name VARCHAR(128) NOT NULL,
    score NUMERIC(5,2) NOT NULL,
    weight NUMERIC(5,2) NOT NULL,
    confidence NUMERIC(5,2) NOT NULL DEFAULT 0,
    reasoning TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    gaps JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (analysis_job_id, dimension_key)
);

-- statement-breakpoint

CREATE INDEX analysis_dimension_scores_job_idx ON analysis_dimension_scores (analysis_job_id);

-- statement-breakpoint

ALTER TABLE analysis_jobs ADD COLUMN overall_score NUMERIC(5,2) NULL;

-- statement-breakpoint

ALTER TABLE analysis_jobs ADD COLUMN verdict VARCHAR(64) NOT NULL DEFAULT '';
