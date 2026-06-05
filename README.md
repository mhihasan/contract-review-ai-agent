# Contract Review

A CLI tool that reads a PDF contract, scores every clause for risk, and produces a structured report telling you what to sign, what to renegotiate, and what to reject. Optionally pauses for a human review step before generating the final output.

---

## What problem it solves

Reading a contract carefully takes hours. High-risk clauses — uncapped liability, one-sided termination rights, ambiguous IP assignments — are easy to miss when buried in dense legal language. This tool reads every clause, flags what looks dangerous, explains why, and tells you what to do about it. You get a signed-off report in minutes, not hours.

---

## What you get

Drop in a PDF. The tool produces a `summary_<contract_id>.md` report with five sections:

**1. Executive Summary**
Two to four sentences on the overall risk profile and whether you should sign.

**2. Signing Recommendation**
One of three verdicts — *Do Not Sign*, *Sign With Changes*, or *Sign As-Is* — with a one-sentence explanation.

**3. Priority Issues**
Every high-risk clause listed with its recommended edit. If three or more medium-risk clauses exist, the section flags them as a cluster that warrants negotiation even if none individually is critical.

**4. Risk Breakdown**
Clause counts by risk level (high / medium / low) and by reviewer decision (approved / rejected / overrides).

**5. Clause-by-Clause Detail**
A table of every clause: risk level, reviewer decision, the issue found, and the recommended edit. Rejected clauses include the reviewer's annotation. Clauses with no recommended edit say "Negotiation required" rather than leaving the cell blank.

The report is written from the reviewing party's perspective — client or vendor. If you provide governing law or contract type, those are included.

---

## Pipeline

```
PDF → extract text → split clauses → analyze each clause → [human review] → summary report
```

---

## Features

### Reads the contract as a whole, not just each clause in isolation

When analyzing a clause, the tool can look up defined terms elsewhere in the document, pull related sections by reference number (e.g. "Section 7.2"), and compare the clause against standard baseline language for its type. A liability clause gets compared to a standard liability cap. An indemnity clause gets compared to a standard mutual indemnity. The analysis is grounded in what the contract actually says, not just the clause text in isolation.

### Three-tier risk scoring with ambiguity detection

Every clause gets a risk level — `high`, `medium`, or `low` — with a plain-language explanation of what the risk is and why it matters. If the clause contains vague or undefined language ("reasonable efforts", "material breach"), that language is quoted verbatim in the finding.

### Human review step

The pipeline can pause after analysis so a reviewer can go clause by clause:

- `approve` — accept it as-is
- `reject` — flag it for renegotiation, with an optional note

Reviewer decisions carry through to the report. The report tracks *overrides* — a reviewer approving a high-risk clause or rejecting a low-risk one — and surfaces them in the Risk Breakdown section. This creates a clear audit trail of what was flagged and what was signed off.

### Concurrent analysis with cost guardrails

All clauses are analyzed in parallel up to a configurable concurrency limit. A shared budget tracks total token usage, dollar cost, and step count across every clause run. If any limit is hit, analysis stops before the next clause starts — no runaway spend on large contracts. Cost and token totals are printed at the end of every run.

### Resumable runs

Every agent step is persisted to the database as it runs. If a run is interrupted — a crash, a timeout, a kill signal — already-finished clauses are not re-analyzed. The pipeline resumes from where it left off. Re-running `summarize` on a completed contract returns the stored result without making another LLM call.

### Dry run

Before committing to a full analysis, `--dry-run` prints the execution plan: clause count, concurrency, per-agent step limit, and projected cost ceiling. No API calls, no database writes.

---

## How the agent works (for developers)

Each clause is processed by a multi-step LLM agent, not a single prompt. The agent runs a tool-calling loop — it can call tools, inspect the results, call more tools, and repeat — before it is required to submit a structured finding. This matters because a liability clause that cross-references Section 4.1 and uses a term defined in Section 1 cannot be assessed accurately from its text alone.

**Tools available to the agent:**

| Tool | What it does |
|---|---|
| `get_definition` | Looks up a defined term in the contract by searching for `"term" means` patterns |
| `get_contract_section` | Fetches a section by name or number (e.g. "Section 7.2") |
| `search_clause_library` | Keyword search over a standard clause library to identify clause type |
| `lookup_standard_clause` | Retrieves full baseline text for a clause type (liability, indemnity, termination, etc.) for comparison |
| `submit_finding` | Structured output gate — the agent cannot finish without calling this with a valid risk level, explanation, and recommendations |

**Context window management**

On long contracts, the agent's message history can grow large. A `ContextManager` monitors token count at each step. When it approaches the compaction threshold it: (1) truncates verbose tool results, (2) summarizes the middle of the history with a separate LLM call, (3) drops the middle entirely as a last resort. The system prompt and most recent messages are always preserved.

**Budget enforcement**

A `Budget` object is shared across all concurrent clause agents. It tracks tokens, USD cost, and step count under a mutex. Before each LLM call, the agent checks whether the budget is already exceeded and exits early if so. This means cost limits are enforced per-step, not just at the end of a run.

**Execution tracing**

Every message and tool call in an agent run is stored in `agent_steps`. The `trace <clause_id>` command replays the full trajectory in the terminal — useful for debugging why the agent reached a particular finding or how many steps it took to get there.

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
| `agent_runs` | One record per clause analysis run — status, step count, token usage, cost. |
| `agent_steps` | Every message and tool call in an agent run, persisted step-by-step. |
| `clause_library` | Standard baseline texts used for clause comparison. |

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
