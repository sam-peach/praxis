CREATE TABLE documents (
    id               TEXT PRIMARY KEY,
    organization_id  TEXT NOT NULL,
    filename         TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'uploaded',
    bom_rows         JSONB NOT NULL DEFAULT '[]',
    warnings         JSONB NOT NULL DEFAULT '[]',
    cloned_from_id   TEXT,
    uploaded_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX documents_org_status_idx ON documents(organization_id, status);
