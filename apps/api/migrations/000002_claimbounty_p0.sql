-- +goose Up
CREATE TABLE email_challenges (
    id uuid PRIMARY KEY,
    subject_id uuid NOT NULL,
    email_ciphertext bytea NOT NULL,
    email_lookup_hash bytea NOT NULL CHECK (octet_length(email_lookup_hash) = 32),
    audience text NOT NULL CHECK (audience IN ('submitter', 'administrator')),
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    expires_at timestamptz NOT NULL,
    attempts_remaining smallint NOT NULL CHECK (attempts_remaining BETWEEN 0 AND 5),
    used_at timestamptz
);
CREATE INDEX email_challenges_lookup_idx ON email_challenges (email_lookup_hash, audience, expires_at DESC);

CREATE TABLE claimbounty_sessions (
    id uuid PRIMARY KEY,
    subject_id uuid NOT NULL,
    email_ciphertext bytea NOT NULL,
    email_lookup_hash bytea NOT NULL CHECK (octet_length(email_lookup_hash) = 32),
    audience text NOT NULL CHECK (audience IN ('submitter', 'administrator')),
    authorization_policy_version varchar(100) NOT NULL,
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    csrf_hash bytea NOT NULL CHECK (octet_length(csrf_hash) = 32),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz
);
CREATE INDEX claimbounty_sessions_email_lookup_idx ON claimbounty_sessions (email_lookup_hash);

CREATE TABLE claimbounty_rate_limits (
    scope varchar(50) NOT NULL,
    key_hash bytea NOT NULL CHECK (octet_length(key_hash) = 32),
    window_started_at timestamptz NOT NULL,
    request_count integer NOT NULL CHECK (request_count > 0),
    PRIMARY KEY (scope, key_hash)
);

CREATE TABLE claimbounty_orders (
    id uuid PRIMARY KEY,
    subject_id uuid NOT NULL,
    submitter_email_ciphertext bytea NOT NULL,
    submitter_email_lookup_hash bytea NOT NULL CHECK (octet_length(submitter_email_lookup_hash) = 32),
    public_reference varchar(15) NOT NULL UNIQUE CHECK (public_reference ~ '^CB-[A-Z0-9]{12}$'),
    status text NOT NULL CHECK (status IN ('draft','awaiting_email_verification','uploading','submitted','scanning','needs_information','ready_for_export','exported','rejected','cancelled','expired')),
    version bigint NOT NULL CHECK (version BETWEEN 1 AND 9007199254740991),
    title varchar(300) NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 300),
    purpose varchar(1000) NOT NULL CHECK (length(btrim(purpose)) BETWEEN 1 AND 1000),
    target_claim_text varchar(5000) NOT NULL CHECK (length(btrim(target_claim_text)) BETWEEN 1 AND 5000),
    target_claim_location varchar(1000) NOT NULL CHECK (length(target_claim_location) <= 1000),
    execute_supplied_code boolean NOT NULL,
    external_search boolean NOT NULL,
    uploads_authorized boolean NOT NULL DEFAULT false,
    analysis_use_authorized boolean NOT NULL DEFAULT false,
    external_redistribution_authorized boolean NOT NULL DEFAULT false CHECK (external_redistribution_authorized = false),
    customer_authorized_at timestamptz,
    contains_participant_data boolean NOT NULL,
    contains_direct_identifiers boolean NOT NULL,
    terms_version varchar(100),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    submitted_at timestamptz,
    retention_policy_version varchar(100) NOT NULL DEFAULT 'intake-30d-v1',
    retention_disposition text NOT NULL DEFAULT 'hard_delete' CHECK (retention_disposition IN ('hard_delete','irreversible_anonymize')),
    source_retention_expires_at timestamptz NOT NULL DEFAULT (now() + interval '30 days'),
    source_deleted_at timestamptz,
    retention_expires_at timestamptz NOT NULL DEFAULT (now() + interval '30 days')
);
CREATE INDEX claimbounty_orders_subject_id_id_idx ON claimbounty_orders (subject_id, id);
CREATE INDEX claimbounty_orders_email_lookup_idx ON claimbounty_orders (submitter_email_lookup_hash);
CREATE INDEX claimbounty_orders_queue_idx ON claimbounty_orders (created_at DESC, id DESC);
CREATE INDEX claimbounty_orders_retention_idx ON claimbounty_orders (retention_expires_at) WHERE retention_expires_at IS NOT NULL;
CREATE INDEX claimbounty_orders_source_retention_idx ON claimbounty_orders (source_retention_expires_at) WHERE source_deleted_at IS NULL;

CREATE TABLE claimbounty_files (
    id uuid PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES claimbounty_orders(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('primary_paper','supplement','preregistration','data','code','environment','data_dictionary','other_evidence')),
    original_display_name varchar(255) NOT NULL CHECK (length(original_display_name) > 0 AND original_display_name !~ '[/\\]'),
    size_bytes bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 262144000),
    sha256 char(64) NOT NULL CHECK (sha256 ~ '^[a-f0-9]{64}$'),
    declared_media_type varchar(255) NOT NULL,
    detected_media_type varchar(255),
    status text NOT NULL CHECK (status IN ('upload_pending','uploaded','scanning','clean','rejected','expired')),
    rejection_code varchar(100),
    storage_key varchar(512) NOT NULL UNIQUE,
    storage_etag varchar(200),
    object_generation varchar(200),
    scanned_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    CHECK ((status = 'upload_pending') OR (storage_etag IS NOT NULL AND object_generation IS NOT NULL)),
    CHECK (role <> 'primary_paper' OR size_bytes <= 52428800)
);
CREATE INDEX claimbounty_files_order_id_idx ON claimbounty_files (order_id, created_at, id);
CREATE INDEX claimbounty_files_scan_idx ON claimbounty_files (created_at, id) WHERE status = 'uploaded';

CREATE TABLE claimbounty_intakes (
    order_id uuid PRIMARY KEY REFERENCES claimbounty_orders(id) ON DELETE CASCADE,
    audit_request jsonb NOT NULL,
    scientific_policy jsonb NOT NULL,
    execution_policy jsonb NOT NULL,
    routine_revision varchar(71) NOT NULL CHECK (routine_revision ~ '^sha256:[a-f0-9]{64}$'),
    routine_validated_at timestamptz NOT NULL,
    routine_evidence_sha256 char(64) NOT NULL CHECK (routine_evidence_sha256 ~ '^[a-f0-9]{64}$'),
    frozen_by uuid NOT NULL,
    frozen_at timestamptz NOT NULL
);

CREATE TABLE claimbounty_exports (
    id uuid PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES claimbounty_orders(id) ON DELETE CASCADE,
    status text NOT NULL CHECK (status IN ('queued','building','ready','failed','expired')),
    routine_id text NOT NULL CHECK (routine_id = 'claim-bounty-operations/run-claimbounty-scientific-audit'),
    routine_revision varchar(71) NOT NULL CHECK (routine_revision ~ '^sha256:[a-f0-9]{64}$'),
    routine_validated_at timestamptz NOT NULL,
    routine_evidence_sha256 char(64) NOT NULL CHECK (routine_evidence_sha256 ~ '^[a-f0-9]{64}$'),
    retention_policy_version varchar(100) NOT NULL,
    preserve_run_outputs boolean NOT NULL,
    sha256 char(64),
    size_bytes bigint CHECK (size_bytes > 0),
    storage_key varchar(512) NOT NULL UNIQUE,
    object_generation varchar(200),
    failure_code varchar(100),
    created_at timestamptz NOT NULL,
    completed_at timestamptz
);
CREATE INDEX claimbounty_exports_queue_idx ON claimbounty_exports (created_at, id) WHERE status = 'queued';
CREATE UNIQUE INDEX claimbounty_exports_one_active_idx ON claimbounty_exports (order_id) WHERE status IN ('queued','building','ready');

CREATE TABLE claimbounty_order_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id uuid NOT NULL REFERENCES claimbounty_orders(id) ON DELETE CASCADE,
    actor_kind text NOT NULL CHECK (actor_kind IN ('submitter','administrator','system')),
    actor_id varchar(255) NOT NULL,
    event_type varchar(100) NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL
);
CREATE INDEX claimbounty_order_events_order_idx ON claimbounty_order_events (order_id, created_at, id);

CREATE TABLE claimbounty_idempotency (
    actor_id uuid NOT NULL,
    operation varchar(50) NOT NULL,
    idempotency_key varchar(128) NOT NULL,
    request_hash bytea NOT NULL CHECK (octet_length(request_hash) = 32),
    order_id uuid REFERENCES claimbounty_orders(id) ON DELETE CASCADE,
    file_id uuid REFERENCES claimbounty_files(id) ON DELETE CASCADE,
    export_id uuid REFERENCES claimbounty_exports(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, operation, idempotency_key)
);

CREATE TABLE claimbounty_outbox (
    id bigserial PRIMARY KEY,
    kind text NOT NULL CHECK (kind IN ('inspect_file','build_export','delete_object')),
    order_id uuid REFERENCES claimbounty_orders(id) ON DELETE CASCADE,
    file_id uuid REFERENCES claimbounty_files(id) ON DELETE CASCADE,
    export_id uuid REFERENCES claimbounty_exports(id) ON DELETE CASCADE,
    storage_key varchar(512),
    object_generation varchar(200),
    retention_order_id uuid,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','done','failed')),
    available_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    failure_code varchar(100),
    CHECK ((kind = 'inspect_file' AND order_id IS NOT NULL AND file_id IS NOT NULL AND export_id IS NULL AND storage_key IS NULL AND retention_order_id IS NULL) OR (kind = 'build_export' AND order_id IS NOT NULL AND export_id IS NOT NULL AND file_id IS NULL AND storage_key IS NULL AND retention_order_id IS NULL) OR (kind = 'delete_object' AND order_id IS NULL AND file_id IS NULL AND export_id IS NULL AND storage_key IS NOT NULL AND object_generation IS NOT NULL))
);
CREATE INDEX claimbounty_outbox_pending_idx ON claimbounty_outbox (available_at, id) WHERE status = 'pending';
CREATE INDEX claimbounty_outbox_retention_delete_idx ON claimbounty_outbox (retention_order_id, status) WHERE kind = 'delete_object' AND retention_order_id IS NOT NULL;

CREATE TABLE claimbounty_tombstones (
    order_id uuid PRIMARY KEY,
    final_status text NOT NULL CHECK (final_status IN ('exported','rejected','cancelled','expired')),
    erased_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE claimbounty_tombstones;
DROP TABLE claimbounty_outbox;
DROP TABLE claimbounty_idempotency;
DROP TABLE claimbounty_order_events;
DROP TABLE claimbounty_exports;
DROP TABLE claimbounty_intakes;
DROP TABLE claimbounty_files;
DROP TABLE claimbounty_orders;
DROP TABLE claimbounty_rate_limits;
DROP TABLE claimbounty_sessions;
DROP TABLE email_challenges;
