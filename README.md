# Contract Review AI Agent

An AI-powered pipeline that extracts clauses from a PDF contract, analyzes each for risk, supports human review, and generates a professional markdown summary report.

## Prerequisites

- Go 1.25+
- PostgreSQL database
- OpenAI or Anthropic API key

Set these environment variables (or create a `.env` file):

```
DATABASE_URL=postgres://user:password@localhost:5432/dbname
LLM_PROVIDER=openai          # or anthropic
OPENAI_API_KEY=sk-...        # if using openai
ANTHROPIC_API_KEY=sk-ant-... # if using anthropic
LLM_MODEL=gpt-4o-mini        # model to use
```

## Usage

### Full automated pipeline (no human review)

```bash
go run . process path/to/contract.pdf
```

Runs the complete pipeline and writes a `summary_<contract_id>.md` file.

### Full pipeline with human review step

```bash
go run . process path/to/contract.pdf --review
```

After analysis completes the pipeline pauses. Run the review and resume commands below.

### Step-by-step

#### 1. Review clauses interactively

```bash
go run . review <contract_id>
```

For each clause you will be prompted to `approve` or `reject` it, with an optional annotation. Press Enter to skip a clause.

#### 2. Complete review and generate summary

```bash
go run . resume <contract_id>
```

Marks the review complete and generates the summary report.

#### 3. Generate (or re-generate) the summary directly

```bash
go run . summarize <contract_id>
```

Generates the final summary for a `review_complete` contract. Safe to run multiple times — re-running on a finished contract prints the existing summary without calling the LLM again.

### Debug commands

```bash
go run . extract <path/to/contract.pdf>          # PDF text extraction only
go run . extract-clauses <contract_id>           # clause splitting only
go run . analyze <contract_id>                   # AI analysis across all clauses
go run . analyze-clause <contract_id> <clause_id> # run agent on a single clause
go run . status <contract_id>                    # show contract and clause processing state
```

### Output

A successful run produces:
- `summary_<contract_id>.md` — a markdown report with five sections: Executive Summary, Signing Recommendation, Priority Issues, Risk Breakdown, and Clause-by-Clause Detail
- The same content printed to stdout

---

## Data Model

### Why each table exists

| Table | Purpose |
|---|---|
| `contracts` | The uploaded document. Tracks the raw text and processing status as it moves through the pipeline. |
| `clauses` | Individual clauses extracted from a contract. A contract is broken into clauses so each can be analyzed independently. |
| `clause_analyses` | The AI's finding for a single clause — risk level, explanation, and recommendations. One analysis per clause. |
| `reviews` | A human reviewer's decision on a clause (approved / rejected) with an optional annotation. |
| `summaries` | A single generated summary for the whole contract. One per contract. |

### Entity Relationship Diagram

**Core pipeline tables**

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

**Agent execution tables**

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
| `uploaded` | File received and stored; processing not yet started. |
| `extracting` | Raw text is being extracted from the document. |
| `extracted` | Raw text extraction complete; ready for clause splitting. |
| `analyzing_clauses` | Contract text is being split into individual clauses. |
| `clauses_extracted` | Clauses saved to the database; ready for AI analysis. |
| `analyzing` | AI is analyzing each clause for risk and ambiguity. |
| `analyzed` | All clause analyses saved; ready for human review. |
| `review_pending` | Waiting for a human reviewer to approve or reject clauses. |
| `review_complete` | All clauses reviewed; ready for summary generation. |
| `summarizing` | Summary of the full contract is being generated. |
| `done` | Pipeline complete; summary available. |
