# Contract Review

A CLI tool that reads a PDF contract, splits it into clauses, scores each one for risk, and produces a structured markdown report. Optionally pauses for a human review step before generating the final output.

## Pipeline

```
PDF → extract text → split clauses → analyze each clause → [human review] → summary report
```

Each clause gets a risk level, an explanation of any concerns, and concrete recommendations. The final report has five sections: Executive Summary, Signing Recommendation, Priority Issues, Risk Breakdown, and Clause-by-Clause Detail.

Output is written to `summary_<contract_id>.md` and printed to stdout.

---

## Getting started

### Prerequisites

- Go 1.25+
- PostgreSQL
- OpenAI or Anthropic API key

### Environment

Create a `.env` file (or export directly):

```
DATABASE_URL=postgres://user:password@localhost:5432/dbname
LLM_PROVIDER=openai          # or anthropic
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
LLM_MODEL=gpt-4o-mini
```

---

## Usage

### Full pipeline

```bash
go run . process path/to/contract.pdf
```

Runs end-to-end and writes `summary_<contract_id>.md`.

### With human review

```bash
go run . process path/to/contract.pdf --review
```

Pauses after analysis. For each clause you'll be prompted to `approve` or `reject` it with an optional annotation. Press Enter to skip.

```bash
go run . review <contract_id>   # step through clauses interactively
go run . resume <contract_id>   # mark review done and generate the summary
```

### Regenerate the summary

```bash
go run . summarize <contract_id>
```

Idempotent — returns the existing summary without re-running analysis if one already exists.

### Dry run

```bash
go run . analyze <contract_id> --dry-run
```

Prints the execution plan (clause count, concurrency, cost estimate). No API calls, no DB writes.

---

## Other commands

| Command | Purpose |
|---|---|
| `extract <path>` | PDF text extraction only |
| `extract-clauses <contract_id>` | Clause splitting only |
| `analyze <contract_id>` | Analyze all clauses |
| `analyze-clause <contract_id> <clause_id>` | Analyze a single clause |
| `status <contract_id>` | Show contract and per-clause state |
| `trace <clause_id>` | Print the step-by-step execution trace for a clause |

---

## Data model

| Table | Purpose |
|---|---|
| `contracts` | The uploaded document and its processing status. |
| `clauses` | Individual clauses extracted from the contract. |
| `clause_analyses` | Risk findings per clause — risk level, explanation, recommendations. |
| `reviews` | Reviewer decisions (approved / rejected) with optional annotations. |
| `summaries` | The final report, one per contract. |

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
        text status "submitted | failed"
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
uploaded → extracting → extracted → analyzing_clauses → clauses_extracted
→ analyzing → analyzed → review_pending → review_complete → summarizing → done
```

| Status | Meaning |
|---|---|
| `uploaded` | File received; processing not yet started. |
| `extracting` | Raw text being extracted from the PDF. |
| `extracted` | Extraction complete; ready for clause splitting. |
| `analyzing_clauses` | Contract text being split into individual clauses. |
| `clauses_extracted` | Clauses saved; ready for analysis. |
| `analyzing` | Analyzing each clause for risk and ambiguity. |
| `analyzed` | All analyses saved; ready for human review. |
| `review_pending` | Waiting for a human reviewer. |
| `review_complete` | All clauses reviewed; ready for summary generation. |
| `summarizing` | Summary being generated. |
| `done` | Pipeline complete; summary available. |
