# ESDD Question Flows

This document details the three primary ESDD (Event-Spec-Driven Development) question flows for developers and AI agents using `specdag`.

---

## 1. Single Feature Flow

Use this when designing or modifying a single vertical feature.

### Question Sequence

1. **What is the human intent?**
   - What value does this feature deliver to the user?
   - Who is the user persona, and what is their entry point?
2. **What expectations define success or failure?**
   - What must happen? (Happy path)
   - What must _never_ happen? (Safety boundaries, invariant constraints)
3. **What evidence exists?**
   - Are there existing specs, code behaviors, database schemas, or screenshots that anchor this feature?
4. **What events, commands, or contracts belong to this feature?**
   - Events (past tense: `x.created`, `y.approved`)
   - Commands (actions: `create.x`, `approve.y`)
   - Contracts (schemas: `x.config.v1`)
5. **What jobs, artifacts, verifiers, or approvals are needed?**
   - Jobs: Computations or background processors
   - Artifacts: Intermediate files, database records, compiled assets
   - Verifiers: Test suites, linters, risk engines, policy checkers
   - Approvals: Human-in-the-loop gates
6. **What must the agent NOT guess?**
   - Which parameters require explicit user configuration?
   - Which architectural patterns are fixed?
7. **Is a local dependency map required?**
   - If severity is Level 2+, create a local `dependency-map.yaml` next to `spec.md`.
8. **What checks prove the feature works correctly?**
   - Unit tests, integration tests, contract compliance checks, or manual approvals.

### Required Artifacts

- `spec.md` (Intent and expectations)
- `tasks.md` (Implementation check-steps)
- `dependency-map.yaml` (Local Spec-DAG, Level 2+)
- `event-flow.md` (Prose event sequence, Level 2+)

---

## 2. Multi-Feature Integration Flow

Use this when multiple features communicate across boundaries via shared events, contracts, or approvals.

### Question Sequence

1. **Which local feature maps are involved?**
   - What are the feature paths (e.g., `specs/features/012-agent-run`, `specs/features/020-bot-activation`)?
2. **Which events or contracts connect them?**
   - Which event is published by feature A and consumed by feature B?
   - Which contract defines the payload of the communication?
3. **Who produces and who consumes?**
   - Verify producer and consumer fields in event/contract catalogs.
4. **Are there node naming conflicts?**
   - Do different features define the same node ID with conflicting titles or types?
5. **Are there global cycles?**
   - Does feature A trigger feature B, which triggers feature C, which loops back to feature A?
6. **What is the downstream impact of a change?**
   - If feature A's event structure changes, which downstream nodes in features B and C are affected?
7. **Which durable truth needs syncing?**
   - Ensure shared events and contracts are moved into global catalogs under `specs/events/` and `specs/contracts/`.

### Validation Tools

- `specdag assemble <dir>` (Detects cycles and naming conflicts)
- `specdag impact <file> <node-id>` (Calculates downstream reachability)
- `specdag check-catalogs <dir>` (Cross-checks against event/contract catalog frontmatter)
- `specdag report <dir>` (Generates unified HTML report highlighting gaps)

---

## 3. Reconciliation Flow

Use this when existing code, legacy specifications, runtime logs, or code knowledge graphs (e.g., GitNexus) disagree with the intended Spec-DAG.

### Question Sequence

1. **What does the Spec-DAG intend?**
   - Review the normative `dependency-map.yaml`. What is the desired behavior and obligation model?
2. **What does the code actually implement?**
   - Analyze the observed codebase using code intelligence graphs (e.g., GitNexus) or AST call-graphs.
3. **What does the runtime execute?**
   - Check logs, traces, or runtime events (`agent_run_trace`).
4. **Where are the mismatches?**
   - Missing implementation (Spec calls for a job, but no code exists).
   - Undeclared dependency (Code calls service B, but the Spec-DAG does not declare it).
   - Missing verifier (Expectation exists in spec, but no test/verifier exists in code).
5. **Is the code wrong, the spec wrong, or is a decision pending?**
   - Do not silently rewrite the spec to match incorrect code.
   - Do not refactor code to match an outdated spec.
6. **What evidence justifies the reconciliation?**
   - Capture log payloads, AST paths, or benchmark results.
7. **What decision is accepted?**
   - Record the outcome in `decisions.md` before updating either the spec or the codebase.

### Core Rule of Reconciliation

> **Never vibe-code to resolve mismatches.**
> Record: `Evidence -> Implication -> Open Gap` first. Decide on the reconciliation path, document it, and then implement the change.
