CREATE TABLE contracts (
    id          text PRIMARY KEY,
    filename    text NOT NULL,
    raw_text    text NOT NULL,
    status      text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE clauses (
    id              text PRIMARY KEY,
    contract_id     text NOT NULL REFERENCES contracts(id),
    sequence_number int  NOT NULL,
    text            text NOT NULL,
    UNIQUE (contract_id, sequence_number)
);

CREATE INDEX idx_clauses_contract_id ON clauses(contract_id);

CREATE TABLE clause_analyses (
    id                  text PRIMARY KEY,
    clause_id           text NOT NULL REFERENCES clauses(id),
    risk_level          text,
    explanation         text,
    ambiguous_language  text,
    recommendations     text,
    status              text NOT NULL DEFAULT 'ok',
    CONSTRAINT chk_clause_analyses_status
        CHECK (status IN ('ok', 'failed')),
    CONSTRAINT chk_clause_analyses_risk_level
        CHECK (status = 'failed' OR risk_level IN ('high', 'medium', 'low'))
);

CREATE INDEX idx_clause_analyses_clause_id ON clause_analyses(clause_id);

CREATE TABLE reviews (
    id          text PRIMARY KEY,
    clause_id   text NOT NULL REFERENCES clauses(id),
    decision    text NOT NULL,
    annotation  text,
    CONSTRAINT chk_reviews_decision
        CHECK (decision IN ('approved', 'rejected'))
);

CREATE INDEX idx_reviews_clause_id ON reviews(clause_id);

CREATE TABLE summaries (
    id          text PRIMARY KEY,
    contract_id text NOT NULL REFERENCES contracts(id) UNIQUE,
    content     text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
