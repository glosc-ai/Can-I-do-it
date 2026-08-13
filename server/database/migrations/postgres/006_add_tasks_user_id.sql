ALTER TABLE tasks ADD COLUMN user_id BIGINT REFERENCES users(id) ON DELETE CASCADE;

-- statement-breakpoint

-- Back-fill NULL rows (tasks created before this migration) so they are
-- owned by the first owner account. An index is added to support the
-- per-user list query efficiently.
UPDATE tasks SET user_id = (SELECT id FROM users WHERE role = 'owner' LIMIT 1)
WHERE user_id IS NULL;

-- statement-breakpoint

ALTER TABLE tasks ALTER COLUMN user_id SET NOT NULL;

-- statement-breakpoint

DROP INDEX IF EXISTS idx_tasks_created_at;

-- statement-breakpoint

CREATE INDEX idx_tasks_user_created ON tasks (user_id, created_at DESC);
