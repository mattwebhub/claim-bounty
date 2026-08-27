-- name: CreateProject :exec
INSERT INTO projects (id, name, created_at, updated_at)
VALUES ($1, $2, $3, $4);

-- name: GetProject :one
SELECT id, name, created_at, updated_at
FROM projects
WHERE id = $1;

-- name: ListProjects :many
SELECT id, name, created_at, updated_at
FROM projects
WHERE NOT sqlc.arg(has_cursor)::boolean
   OR (created_at, id) < (sqlc.arg(cursor_created_at)::timestamptz, sqlc.arg(cursor_id)::uuid)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit);
