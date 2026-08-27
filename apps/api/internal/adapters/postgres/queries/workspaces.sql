-- name: CreateWorkspace :exec
INSERT INTO workspaces (project_id, document, version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetWorkspace :one
SELECT project_id, document, version, created_at, updated_at
FROM workspaces
WHERE project_id = $1;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET document = sqlc.arg(document),
    version = sqlc.arg(new_version),
    updated_at = sqlc.arg(updated_at)
WHERE project_id = sqlc.arg(project_id)
  AND version = sqlc.arg(expected_version)
RETURNING project_id, document, version, created_at, updated_at;
