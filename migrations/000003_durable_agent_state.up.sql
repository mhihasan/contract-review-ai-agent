CREATE TABLE runs (
    id           text PRIMARY KEY,
    contract_id  text NOT NULL REFERENCES contracts(id),
    started_at   timestamptz NOT NULL DEFAULT now(),
    ended_at     timestamptz,
    status       text NOT NULL,
    reached_stage text
);

CREATE TABLE agent_runs (
    id           text PRIMARY KEY,
    clause_id    text NOT NULL REFERENCES clauses(id),
    run_id       text REFERENCES runs(id),
    status       text NOT NULL,
    step_count   int  NOT NULL DEFAULT 0,
    used_tokens  int  NOT NULL DEFAULT 0,
    used_cost_usd numeric NOT NULL DEFAULT 0,
    started_at   timestamptz NOT NULL DEFAULT now(),
    ended_at     timestamptz
);

CREATE TABLE agent_steps (
    id            text PRIMARY KEY,
    agent_run_id  text NOT NULL REFERENCES agent_runs(id),
    step_index    int  NOT NULL,
    messages_json jsonb NOT NULL,
    usage_json    jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (agent_run_id, step_index)
);
