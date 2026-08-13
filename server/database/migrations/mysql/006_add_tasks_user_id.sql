ALTER TABLE tasks ADD COLUMN user_id BIGINT UNSIGNED NULL,
    ADD CONSTRAINT fk_tasks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- statement-breakpoint

UPDATE tasks SET user_id = (SELECT id FROM users WHERE role = 'owner' LIMIT 1)
WHERE user_id IS NULL;

-- statement-breakpoint

ALTER TABLE tasks MODIFY COLUMN user_id BIGINT UNSIGNED NOT NULL;

-- statement-breakpoint

ALTER TABLE tasks DROP INDEX idx_tasks_created_at;

-- statement-breakpoint

ALTER TABLE tasks ADD INDEX idx_tasks_user_created (user_id, created_at DESC);
