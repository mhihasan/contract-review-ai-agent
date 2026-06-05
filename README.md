# AI Contract Reviewer

A working Go implementation of a contract review agent, built to explore how to engineer LLM agents that are resumable, cost-bounded, and structured in their output. The contract domain is real and useful; the engineering patterns are the point.

---

## What you will find here

- Forced structured output: the agent can only finish by calling a dedicated tool with validated fields. There is no prose exit path.
- Per-step durable persistence: every agent step is written to the database as it runs. Crashes do not lose work.
- Resume on crash: a restarted run picks up from the last completed step, not from the beginning.
- Shared budget enforcement: tokens, dollars, and steps are tracked across all concurrent agents. One limit stops everything.
- Context window management: tool results are truncated first, then the middle of history is summarized, then dropped if needed. The system prompt and recent turns are never evicted.
- Idempotent pipeline stages: every stage checks whether it has already run. Re-running the pipeline never re-bills.
- Provider-agnostic LLM interface: swap between OpenAI and Anthropic via config. Cost estimation stays accurate either way.
- Dry-run mode: prints what the pipeline would do without making any API calls or database writes.

---

## What it does

Point it at a PDF contract and it reads every clause, scores the risk, and writes a report telling you what to sign, what to push back on, and what to reject outright. You can also wire in a human review step before the report gets generated.

Every run produces a markdown report with:

- **Executive Summary**: overall risk and a plain signing recommendation
- **Signing Recommendation**: one verdict: Do Not Sign, Sign With Changes, or Sign As-Is
- **Priority Issues**: every high-risk clause with the recommended edit
- **Risk Breakdown**: clause counts by risk level, reviewer decisions, and overrides
- **Clause-by-Clause Detail**: a table of every clause with risk, issue, and recommended fix

The following is real output from running the tool on `testdata/sample-contract.pdf`:

```
## Executive Summary

The contract presents a high overall risk profile from the client's perspective,
with significant concerns regarding liability limitations, intellectual property
rights, and ambiguity in key terms. There are eight high-risk clauses that could
expose the client to substantial financial and operational risks. It is
recommended to negotiate changes before signing to mitigate these risks.

Signing Recommendation: Sign With Changes
The contract contains multiple high-risk clauses that require negotiation to
protect the client's interests and ensure clarity in obligations and rights.

## Priority Issues (excerpt)

Clause 17 - Liability Cap   Risk: High
The clause limits the Vendor's total liability to fees paid in the three months
preceding a claim. Recommended edit: renegotiate the liability cap to align with
the standard twelve-month period to ensure better protection against claims.

Clause 9 - IP Assignment   Risk: High
The clause assigns all rights to work products and inventions created by the
Vendor to the Client, which may limit the Vendor's ability to use or
commercialize their own creations. Recommended edit: clarify definitions of
"work product" and "inventions" and consider allowing the Vendor to retain
rights to inventions with commercial value outside this contract.

## Risk Breakdown
High: 8   Medium: 24   Low: 0
Approved: 0   Rejected: 0   Overrides: 0
```

Full output: [testdata/sample_output/summary_0f6c4821-7afc-4bcf-ba61-3f03f438ade4.md](https://github.com/mhihasan/contract-review-ai-agent/blob/main/testdata/sample_output/summary_0f6c4821-7afc-4bcf-ba61-3f03f438ade4.md)

---

## Quickstart

```bash
git clone https://github.com/mhihasan/contract-review-ai-agent
cd contract-review-ai-agent
cp .env.example .env
docker compose up -d postgres
mise install
make migrate-up
go run . process testdata/sample-contract.pdf
```

A sample contract is included at [`testdata/sample-contract.pdf`](https://github.com/mhihasan/contract-review-ai-agent/blob/main/testdata/sample-contract.pdf) to try immediately.

Fill in `DATABASE_URL` and `OPENAI_API_KEY` in `.env` before running.

`mise` is required. Go 1.25 is pinned in `mise.toml` and is not yet in standard toolchains.

---

## The problem

Nobody reads contracts carefully. There are too many clauses, the language is dense, and the risky parts (uncapped liability, one-sided termination, vague IP assignments) do not announce themselves. By the time a lawyer flags something it is usually in the middle of a signing crunch.

This tool does not replace legal review. But it reads the whole thing, flags what looks bad, explains why it matters, and gives you a draft edit for each problem clause. That is most of the work.

---

## How it works

```
PDF -> extract text -> split into clauses -> analyze each clause -> [human review] -> report
```

Each clause gets its own LLM agent run. The agent does not just read the clause. It can look up defined terms elsewhere in the contract, pull referenced sections by number, and compare the clause against a library of standard baseline texts. A clause that says "as defined in Section 4.1" actually gets that section fetched before the risk is assessed.

**Layer separation:** `pipeline/` sequences the stages and owns contract status. `agent/` runs the tool-use loop for a single clause. `tool/` implements each tool the agent can call. `llm/` abstracts the provider. `store/` owns all persistence. Each layer can be tested and extended independently.

---

## Agentic engineering practices

These are the patterns that make the agent reliable enough to run unattended on long contracts.

**Forced structured output.**
The agent can only finish by calling a dedicated tool with a validated result. Risk level, explanation, and recommendations are all required fields. If the agent attempts to submit an incomplete result, the call is rejected and the loop continues. There is no way to exit with unstructured prose.

**Per-step persistence.**
Each step in an agent run is written to the database immediately after it completes. The full message history is stored at every step, not just at the end. This provides a complete audit trail of what the agent did and why, and makes runs resumable after any failure.

**Resume on crash.**
When a run restarts, it loads the last saved step from the database and continues from there. It does not re-run completed steps or re-bill for work already done. The budget is also restored from saved totals so limits remain accurate across restarts.

**Pre-call budget checks.**
The shared budget is checked before each LLM call, not after. A clause that would exceed the token or cost limit does not start at all. This prevents the state where work is half-done and already billed but nothing was saved.

**Shared concurrent budget.**
All clause agents run in parallel but share a single budget object. One mutex-protected struct tracks total tokens, dollars, and steps across every goroutine. There is no per-agent budget that could collectively exceed the cap.

**Context compaction.**
Rather than letting the context grow until it crashes, the agent manages it actively. First it truncates oversized tool results. If still over the threshold, it summarizes the middle block of history. If still over, it drops the middle entirely. The system prompt and the most recent turns are pinned and never evicted.

**Idempotent pipeline stages.**
Every stage checks the contract's current status before doing any work. If the stage has already run, it does nothing. Re-running the analysis skips clauses that already have a saved finding. Re-running the report step returns the stored report. Nothing re-bills.

**Provider-agnostic LLM interface.**
The agent talks to an abstracted interface, not a specific provider SDK. OpenAI and Anthropic are both implemented and can be swapped via the `LLM_PROVIDER` environment variable. Cost estimation is per-provider so budget enforcement stays accurate regardless of which model is in use.

---

## Usage

### Full pipeline

```bash
go run . process path/to/contract.pdf
```

### With human review

```bash
go run . process path/to/contract.pdf --review
# pipeline pauses after analysis
go run . review <contract_id>
go run . resume <contract_id>
```

### Regenerate report

```bash
go run . summarize <contract_id>
```

If a report already exists, it prints the stored version with no LLM call.

### Estimate cost before running

```bash
go run . analyze <contract_id> --dry-run
```

---

## Commands

| Command | What it does |
|---|---|
| `process <path>` | Run the full pipeline |
| `review <contract_id>` | Step through clauses interactively |
| `resume <contract_id>` | Complete review and advance to report |
| `extract <path>` | PDF text extraction only |
| `extract-clauses <contract_id>` | Clause splitting only |
| `analyze <contract_id>` | Run analysis across all clauses |
| `analyze-clause <contract_id> <clause_id>` | Run agent on one clause |
| `status <contract_id>` | Show contract and per-clause state |
| `trace <clause_id>` | Replay the agent's step-by-step trace |

---

## Environment variables

```
DATABASE_URL=postgres://user:password@localhost:5432/dbname
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
LLM_MODEL=gpt-4o-mini
```

---

## Data model

| Table | What is in it |
|---|---|
| `contracts` | The uploaded document and its current processing status |
| `clauses` | Each clause extracted from the contract, in order |
| `clause_analyses` | The risk finding for each clause: level, explanation, ambiguous language, recommendations |
| `reviews` | Reviewer decisions: approved or rejected, with optional annotation |
| `summaries` | The final report, one per contract |
| `agent_runs` | One record per clause run: status, step count, tokens used, cost |
| `agent_steps` | Every message and tool call in a run, stored as it happens |
| `clause_library` | Standard clause baselines the agent compares against |

### Entity relationships

**Core pipeline**

```mermaid
erDiagram
    contracts {
        text id PK
        text filename
        text raw_text
        text status
        bool requires_review
        timestamptz created_at
    }

    clauses {
        text id PK
        text contract_id FK
        int sequence_number
        text text
    }

    clause_analyses {
        text id PK
        text clause_id FK
        text risk_level "nullable"
        text explanation "nullable"
        text ambiguous_language "nullable"
        text recommendations "nullable"
        text status "analyzed | failed"
    }

    reviews {
        text id PK
        text clause_id FK
        text decision "approved | rejected"
        text annotation "nullable"
    }

    summaries {
        text id PK
        text contract_id FK "unique"
        text content
        timestamptz created_at
    }

    contracts ||--o{ clauses : "has"
    contracts ||--o| summaries : "summarized by"
    clauses ||--o| clause_analyses : "analyzed by"
    clauses ||--o| reviews : "reviewed by"
```

**Agent execution**

```mermaid
erDiagram
    contracts {
        text id PK
    }

    clauses {
        text id PK
        text contract_id FK
    }

    runs {
        text id PK
        text contract_id FK
        text status
        text reached_stage "nullable"
        timestamptz started_at
        timestamptz ended_at "nullable"
    }

    agent_runs {
        text id PK
        text clause_id FK
        text run_id "nullable"
        text status
        int step_count
        int used_tokens
        numeric used_cost_usd
        timestamptz started_at
        timestamptz ended_at "nullable"
    }

    agent_steps {
        text id PK
        text agent_run_id FK
        int step_index
        bytes messages_json
        bytes usage_json
        timestamptz created_at
    }

    clause_library {
        text id PK
        text clause_type
        text standard_text
        text notes
    }

    contracts ||--o{ runs : "has"
    contracts ||--o{ clauses : "has"
    clauses ||--o{ agent_runs : "processed by"
    agent_runs ||--o{ agent_steps : "has"
```

### Contract status flow

```
uploaded -> extracting -> extracted -> analyzing_clauses -> clauses_extracted
-> analyzing -> analyzed -> review_pending -> review_complete -> summarizing -> done
```

| Status | Meaning |
|---|---|
| `uploaded` | File received, nothing started yet |
| `extracting` | Pulling raw text out of the PDF |
| `extracted` | Text ready, waiting for clause splitting |
| `analyzing_clauses` | Splitting the contract text into individual clauses |
| `clauses_extracted` | Clauses saved, ready for analysis |
| `analyzing` | Running the agent on each clause |
| `analyzed` | All clauses done, ready for human review |
| `review_pending` | Waiting on a reviewer |
| `review_complete` | Review done, ready to generate the report |
| `summarizing` | Generating the report |
| `done` | Done. Report is available |

---

## Contributing

Issues and PRs are welcome. Good places to start are adding a new tool, adding a new LLM provider, or extending the clause library.

