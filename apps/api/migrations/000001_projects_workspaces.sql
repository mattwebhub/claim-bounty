-- +goose Up
CREATE TABLE projects (
    id uuid PRIMARY KEY CHECK (id <> '00000000-0000-0000-0000-000000000000'::uuid),
    name varchar(120) NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at)
);

CREATE TABLE workspaces (
    project_id uuid PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    document jsonb NOT NULL CHECK (
        jsonb_typeof(document) = 'object'
        AND document ? 'schemaVersion'
        AND document -> 'schemaVersion' = '1'::jsonb
        AND document ? 'objects'
        AND jsonb_typeof(document -> 'objects') = 'array'
        AND jsonb_array_length(document -> 'objects') <= 1000
    ),
    version bigint NOT NULL CHECK (version BETWEEN 1 AND 9007199254740991),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at)
);

CREATE INDEX projects_created_at_id_idx ON projects (created_at DESC, id DESC);

-- +goose Down
DROP TABLE workspaces;
DROP TABLE projects;
