# Artifact Checklist

Use this checklist when creating or reviewing a feature folder.

## spec.md

- Goal and non-goals are explicit.
- Intent names wanted outcome, constraints, success scenarios, failure
  scenarios, and connected features/domains.
- Expectations define done, failed, must-not-happen, and verifiable checks.
- Agent Guessing Gaps are listed or explicitly marked none.
- User-visible behavior is separated from implementation detail.
- Acceptance criteria are checkable.
- Context/evidence has provenance: file, command, source link, date, or runtime
  artifact.
- Evidence Inventory exists for Level 2+ work.
- Open questions do not block the stated scope.

## event-flow.md

- Bounded context and event owner are named.
- Events are separated from commands, queries, and status snapshots.
- Producer, consumers, schema, causation/correlation, ordering, replay, and
  idempotency are explicit.
- Privacy class and redaction rules are stated where relevant.
- Event names are marked draft until owner and consumers are stable.

## dependency-map.yaml

- Declares a clear `kind` (e.g., `spec_dependency`, `event_flow`, `agent_run_trace`).
- Node types and edge types follow the allowed schema.
- All nodes have unique IDs.
- Graph contains no cycles (if it's a DAG/workflow/trace).
- References to external events or contracts match their respective catalogs.

## vocabulary.md

- Core terms have one preferred name.
- Synonyms are intentional, not accidental aliases.
- Anti-pattern terms are named for collision-prone concepts.
- Commands, queries, and policies are separated.
- Ownership of vocabulary and schema evolution is explicit.

## plan.md

- Affected files/modules/contracts are named.
- Dependency and migration risks are named.
- Verification gates are selected by risk.
- Non-goals from the spec are preserved.

## tasks.md

- Tasks are small and checkable.
- Discovery and validation tasks are included.
- Completed tasks have evidence.

## research.md

- Findings are curated.
- Sources are dated or linked.
- External claims prefer official/primary sources.
- Raw material is referenced, not pasted unnecessarily.

## decisions.md

- Each decision has date, status, decision, evidence, rejected alternatives,
  and follow-up/finalization target.

## contracts/

- Draft vs final status is clear.
- Owners are named.
- Compatibility and failure behavior are described.
- Event/API/auth/link contracts name versioning and migration expectations.

## events/

- Catalog entries have owner, meaning, producer, consumers, schema, and status.
- Flows describe causality, not only transport topology.
- Internal events are not confused with published integration events.

## archive/*/disposition.md

- Status is clear.
- Replacement/superseding feature is linked.
- Reusable evidence and known gaps are listed.
- Unfinished work is routed to future features.

## sync to durable truth

- Durable vocabulary moved to `domains/`.
- Accepted APIs/auth/data contracts moved to `contracts/`.
- Published events/schemas/flows moved to `events/`.
- Cross-feature architecture moved to `foundations/`.
- If sync was skipped, the reason is recorded.
