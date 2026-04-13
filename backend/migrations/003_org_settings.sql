CREATE TABLE org_settings (
    org_id                UUID        PRIMARY KEY REFERENCES organizations(id),
    export_columns        JSONB       NOT NULL DEFAULT '["internalPartNumber","quantity"]',
    export_include_header BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
