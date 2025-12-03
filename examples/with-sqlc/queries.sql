-- name: ListTasks :many
SELECT * FROM tasks
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = ?;

-- name: CreateTask :one
INSERT INTO tasks (title, description)
VALUES (?, ?)
RETURNING *;

-- name: UpdateTask :one
UPDATE tasks
SET title = ?, description = ?, completed = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = ?;

-- name: ListIncompleteTasks :many
SELECT * FROM tasks
WHERE completed = 0
ORDER BY created_at DESC;
