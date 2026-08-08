CREATE TABLE analysis_dimension_scores (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    analysis_job_id BIGINT NOT NULL,
    dimension_key VARCHAR(64) NOT NULL,
    dimension_name VARCHAR(128) NOT NULL,
    score DECIMAL(5,2) NOT NULL,
    weight DECIMAL(5,2) NOT NULL,
    confidence DECIMAL(5,2) NOT NULL DEFAULT 0,
    reasoning TEXT NOT NULL,
    evidence JSON NOT NULL,
    gaps JSON NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT dimension_scores_job_fk FOREIGN KEY (analysis_job_id) REFERENCES analysis_jobs(id) ON DELETE CASCADE,
    UNIQUE KEY analysis_dimension_scores_job_dimension (analysis_job_id, dimension_key),
    INDEX analysis_dimension_scores_job_idx (analysis_job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- statement-breakpoint

ALTER TABLE analysis_jobs ADD COLUMN overall_score DECIMAL(5,2) NULL;

-- statement-breakpoint

ALTER TABLE analysis_jobs ADD COLUMN verdict VARCHAR(64) NOT NULL DEFAULT '';
