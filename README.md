# Contract Review AI Agent

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

```
contracts
│   id (PK)
│   filename
│   raw_text
│   status
│   created_at
│
└──< clauses
        id (PK)
        contract_id (FK → contracts)
        sequence_number
        text
        │
        ├──< clause_analyses
        │       id (PK)
        │       clause_id (FK → clauses)
        │       risk_level       -- nullable (null when status = 'failed')
        │       explanation
        │       ambiguous_language
        │       recommendations
        │       status           -- 'ok' | 'failed'
        │
        └──< reviews
                id (PK)
                clause_id (FK → clauses)
                decision         -- 'approved' | 'rejected'
                annotation

summaries
    id (PK)
    contract_id (FK → contracts)   -- unique (one summary per contract)
    content
    created_at
```

### Contract status flow

```
uploaded → extracting → extracted → analyzing_clauses → clauses_extracted
→ analyzing → analyzed → review_pending → review_complete → summarizing → done
```
