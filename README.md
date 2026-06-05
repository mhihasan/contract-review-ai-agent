# AI Contract Reviewer

Point it at a PDF contract, and it reads every clause, scores the risk, and writes a report telling you what to sign, what to push back on, and what to reject outright. You can also wire in a human review step before the report gets generated.

---

## The problem

Nobody reads contracts carefully. There are too many clauses, the language is dense, and the risky parts — uncapped liability, one-sided termination, vague IP assignments — don't announce themselves. By the time a lawyer flags something it's usually in the middle of a signing crunch.

This tool doesn't replace legal review. But it reads the whole thing, flags what looks bad, explains why it matters, and gives you a draft edit for each problem clause. That's most of the work.

---

## Output

Every run produces a `summary_<contract_id>.md` with five sections:

**Executive Summary** — a few sentences on the overall risk and a plain signing recommendation.

**Signing Recommendation** — one verdict: *Do Not Sign*, *Sign With Changes*, or *Sign As-Is*, with a sentence on why.

**Priority Issues** — every high-risk clause with the recommended edit. If there are three or more medium-risk clauses, it flags them as a group — individually they might be fine, together they're a negotiation problem.

**Risk Breakdown** — counts: how many clauses at each risk level, how many were approved or rejected by a reviewer, and how many overrides there were (a reviewer approving something flagged high, or rejecting something flagged low).

**Clause-by-Clause Detail** — a table of every clause with risk, decision, issue, and the recommended fix. Nothing is left blank — if there's no draft language, it says "Negotiation required."

The report is written from your perspective — client or vendor. Pass in governing law and contract type if you have them.

---

## How it works

```
PDF → extract text → split into clauses → analyze each clause → [human review] → report
```

Each clause gets its own LLM agent run. The agent doesn't just read the clause — it can look up defined terms elsewhere in the contract, pull referenced sections by number, and compare the clause against a library of standard baseline texts (liability caps, indemnity language, termination clauses, etc.). A clause that says "as defined in Section 4.1" actually gets Section 4.1 fetched before the risk is assessed.

The agent has to finish by calling `submit_finding` with a structured result — risk level, explanation, any ambiguous language quoted verbatim, and concrete recommendations. It can't just output prose and call it done.

---

## Features

**Concurrent analysis with cost controls.** Clauses run in parallel. There's a shared budget across all clause agents — tokens, dollars, and steps. If any limit is hit, the next clause doesn't start. No runaway spend on a 60-page MSA. Token and cost totals print at the end of every run.

**Human review step.** Run with `--review` and the pipeline pauses after analysis. You step through each clause and either `approve` it or `reject` it with a note. Overrides — approving something high-risk, rejecting something low-risk — are tracked separately in the report.

**Resumable.** Every agent step is written to the database as it runs. If a run crashes or gets killed, restart it and it picks up from the last finished clause. `summarize` on a completed contract returns the stored report without re-running anything.

**Dry run.** `--dry-run` on `analyze` prints what it would do — clause count, concurrency, step limit, cost ceiling — without touching the LLM or the database.

---

## Inside the agent loop

The agent runs a tool-calling loop, not a one-shot prompt. It calls tools, reads the results, decides what else it needs, and keeps going until it has enough to submit a finding. Max steps is configurable (default 12).

Tools the agent can call:

| Tool | What it does |
|---|---|
| `get_definition` | Finds where a term is defined in the contract (`"X" means ...` patterns) |
| `get_contract_section` | Fetches a section by name or sequence number |
| `search_clause_library` | Keyword search over standard clause templates |
| `lookup_standard_clause` | Gets the full baseline text for a clause type — used to spot deviations |
| `submit_finding` | The only way to finish. Requires risk level, explanation, and recommendations. |

## Agentic engineering practices

- **Structured output via tool forcing.** The agent can only finish by calling `submit_finding`. There's no code path where it outputs prose and exits. Risk level is an enum — `high`, `medium`, `low` — validated at call time. If the agent tries to submit without an explanation or recommendations, the call is rejected and it has to try again.
- **Tool-use loop with bounded steps.** The agent runs a proper tool-calling loop, not a chain of prompts. Each step it decides what it needs, calls a tool, reads the result, and decides what's next. Max steps is capped (default 12) to prevent infinite loops on ambiguous clauses.
- **Context compaction strategy.** Rather than letting the context grow until it crashes, the agent actively manages it: truncate tool results first, then summarize the middle of the history, then drop it if needed. The system prompt and recent turns are pinned and never evicted.
- **Pre-call budget checks.** The shared budget is checked before each LLM call, not after. A clause that would exceed the token or cost limit doesn't start — so you never end up in a state where half a clause was analyzed and billed but nothing was saved.
- **Per-step persistence.** Each step in an agent run is written to the database immediately. This makes runs resumable after any kind of failure — crash, timeout, kill signal. It also means you have a complete audit log of what the agent did and why, not just the final answer.
- **Idempotent pipeline stages.** Every stage checks whether it's already been done before doing work. Re-running `analyze` skips clauses that already have a saved finding. Re-running `summarize` returns the stored report. Nothing re-bills.
- **Parallelism with shared rate control.** Clauses run concurrently, but a single `Budget` object (mutex-protected) coordinates across all goroutines. One place tracks spend, one place enforces limits — no per-agent budget that could collectively blow past the cap.
- **Dry run mode.** The pipeline can print its execution plan — clause count, concurrency, step limit, cost ceiling — without making any API calls or DB writes. Useful for cost estimation before running a large contract.
- **Provider-agnostic LLM interface.** The agent talks to an `LLM` interface, not a specific provider SDK. Swap between OpenAI and Anthropic via config. Cost estimation is per-provider so budget enforcement stays accurate regardless of which model is in use.

---

## Getting started

### Prerequisites

- Go 1.25+
- PostgreSQL
- OpenAI or Anthropic API key

### Environment

```
DATABASE_URL=postgres://user:password@localhost:5432/dbname
LLM_PROVIDER=openai          # or anthropic
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
LLM_MODEL=gpt-4o-mini
```

Put these in a `.env` file or export them directly.

---

## Usage

### Run the full pipeline

```bash
go run . process path/to/contract.pdf
```

### With a human review step

```bash
go run . process path/to/contract.pdf --review
```

Pauses after analysis. Then:

```bash
go run . review <contract_id>   # go through clauses, approve or reject each
go run . resume <contract_id>   # generate the report once you're done
```

### Regenerate the report

```bash
go run . summarize <contract_id>
```

If the report already exists, it prints the stored version. No LLM call.

### See what it would do before running

```bash
go run . analyze <contract_id> --dry-run
```

---

## Other commands

| Command | What it does |
|---|---|
| `extract <path>` | PDF text extraction only |
| `extract-clauses <contract_id>` | Clause splitting only |
| `analyze <contract_id>` | Analyze all clauses |
| `analyze-clause <contract_id> <clause_id>` | Analyze one clause |
| `status <contract_id>` | Show contract and per-clause state |
| `trace <clause_id>` | Replay the agent's step-by-step trace for a clause |

---

## Data model

| Table | What's in it |
|---|---|
| `contracts` | The uploaded document and its current processing status. |
| `clauses` | Each clause extracted from the contract, in order. |
| `clause_analyses` | The risk finding for each clause — level, explanation, ambiguous language, recommendations. |
| `reviews` | Reviewer decisions: approved or rejected, with optional annotation. |
| `summaries` | The final report. One per contract. |
| `agent_runs` | One record per clause run — status, step count, tokens used, cost. |
| `agent_steps` | Every message and tool call in a run, stored as it happens. |
| `clause_library` | Standard clause baselines the agent compares against. |

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
| `uploaded` | File received, nothing started yet. |
| `extracting` | Pulling raw text out of the PDF. |
| `extracted` | Text ready, waiting for clause splitting. |
| `analyzing_clauses` | Splitting the contract text into individual clauses. |
| `clauses_extracted` | Clauses saved, ready for analysis. |
| `analyzing` | Running the agent on each clause. |
| `analyzed` | All clauses done, ready for human review. |
| `review_pending` | Waiting on a reviewer. |
| `review_complete` | Review done, ready to generate the report. |
| `summarizing` | Generating the report. |
| `done` | Done. Report is available. |
