# README Redesign Spec — 2026-06-05

## Goal

Rewrite the README so it works for two readers simultaneously:
1. **AI engineers** studying agentic system design who want a real, honest reference implementation
2. **Developers** who want to run the tool on an actual contract

The engineering is the headline. The contract use case is the concrete vehicle that makes it tangible.

---

## Positioning

- **Honest framing**: built while learning, shared because others might find it useful
- **Serious engineering**: the agentic patterns (persistence, budget enforcement, structured output, context management) are production-quality and worth studying
- **Neither polished product nor toy**: the contract domain is real and useful, but this is not a SaaS — it's a CLI tool and reference codebase

---

## Tone

- Direct, no hype
- "Here is what it does and why the engineering decisions matter"
- Learning-in-public: transparent about trade-offs, doesn't hide the seams
- Hybrid depth style: one sentence on **why** each pattern matters, then a code link for **how**

---

## Structure

### 1. Hook (2 sentences)

Engineering-first. What this is and why it exists.

> A working Go implementation of a contract review agent — built to explore how to engineer LLM agents that are resumable, cost-bounded, and structured in their output. The contract domain is real and useful; the engineering patterns are the point.

### 2. What you'll find here (bullet list, ~8 items)

The "is this worth my time?" scan for AI engineers. Each bullet names one agentic pattern.

- Forced structured output via tool-use exit (no prose leakage)
- Per-step durable persistence — runs survive crashes and resume exactly where they stopped
- Shared budget enforcement across concurrent agents (tokens, dollars, steps)
- Context window management with pinned history and middle-summarization
- Idempotent pipeline stages — nothing re-bills on re-run
- Provider-agnostic LLM interface (OpenAI and Anthropic, swappable via config)
- Dry-run mode — cost estimation without API calls or DB writes
- Concurrent clause analysis with a single mutex-protected budget object

### 3. Sample output (trimmed excerpt, clearly labelled)

~20 lines showing what a real report looks like — executive summary, one high-risk finding with recommendations. Because all `pipeline/summary_*.md` files in the repo are stubs, write a realistic illustrative example, labelled `<!-- illustrative output — run on a sample contract -->`.

### 4. Quickstart (5 commands, no missing steps)

```bash
git clone https://github.com/mhihasan/contract-review-ai-agent
cd contract-review-ai-agent
cp .env.example .env          # fill in DATABASE_URL and OPENAI_API_KEY
docker compose up -d postgres
mise install                   # pins Go 1.25, golangci-lint, sqlc, gomigrate
make migrate-up
go run . process testdata/sample-contract.pdf
```

Note: Go 1.25 is not in standard toolchains yet. `mise` is the intended install path — `mise.toml` pins the exact version.

### 5. How it works

Keep the existing pipeline ASCII diagram. Add one short paragraph explaining the design intent behind the layer separation (`pipeline/`, `agent/`, `tool/`, `llm/`, `store/`) — why each exists and what it protects. This is what engineers who want to extend it need before reading code.

### 6. Agentic engineering practices (promoted, hybrid depth)

Move this section earlier (currently buried after Features and "Inside the agent loop"). Merge the redundant "Inside the agent loop" section into this one.

For each practice: one sentence on **why it matters**, one code link.

Practices to cover (8 total):
1. Forced structured output — `agent/agent.go:125` (`ForceToolName`)
2. Per-step persistence — `agent/agent.go:203` (`AppendAgentStep`)
3. Resume on crash — `agent/agent.go:72` (run state reload)
4. Pre-call budget check — `agent/agent.go:119` (check before LLM call)
5. Shared concurrent budget — `agent/budget.go` (mutex-protected)
6. Context compaction — `agent/context.go`
7. Idempotent pipeline stages — `pipeline/clause_splitting.go` (status guard)
8. Provider-agnostic LLM — `llm/llm.go` (interface) + `llm/factory.go`

### 7. Commands reference

Keep as-is. Already clear.

### 8. Data model

Keep Mermaid diagrams. Fix one bug: `clause_analyses.status` diagram says `submitted | failed` but `agent/agent.go:298` sets `Status: "analyzed"`. Correct the diagram to reflect actual code.

### 9. Built while learning (3 sentences)

Short honest close.

> This started as a project to learn how to build agents that don't break in production. The contract domain turned out to be a good vehicle — enough complexity to expose real edge cases, concrete enough to produce useful output. Issues and PRs are welcome; good places to start are adding a new tool, adding a new LLM provider, or extending the clause library.

---

## What is NOT changing

- The "The problem" section prose is good — keep it but move it below the quickstart, not at the top
- The commands reference table — already clear, no changes
- The contract status flow table — already correct, keep it

---

## Known bug to fix inline

`clause_analyses.status` in the ER diagram currently shows `"submitted | failed"`. The application sets `"analyzed"` (`agent/agent.go:298`). The diagram should show `"analyzed | failed"`. This is noted as a footgun in `AGENTS.md` — the README should not silently perpetuate the mismatch.

---

## Out of scope

- CI badges (no CI configured)
- Splitting into multiple docs
- CONTRIBUTING.md file
- Formal architecture diagrams beyond the existing pipeline ASCII
