CREATE TABLE match_feedback (
    id               TEXT PRIMARY KEY,
    organization_id  TEXT NOT NULL,
    drawing_id       TEXT NOT NULL,
    candidate_id     TEXT NOT NULL,
    action           TEXT NOT NULL,  -- 'accept' | 'reject'
    score            DOUBLE PRECISION NOT NULL,
    score_breakdown  JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX match_feedback_drawing_idx ON match_feedback(drawing_id);
CREATE INDEX match_feedback_org_idx     ON match_feedback(organization_id);
