# ESDD Question Flows

This document details the three primary ESDD (Event-Spec-Driven Development) question flows for developers and AI agents using `specdag`.

---

## 1. Single Feature Flow

### When to use
Use this when designing or modifying a single vertical feature.

### Required artifacts
- `spec.md` (Intent and expectations, with severity defined in frontmatter)
- `tasks.md` (Implementation check-steps)
- `dependency-map.yaml` (Local Spec-DAG, required for Level 2+)
- `event-flow.md` (Prose event sequence, required for Level 2+)

### Questions to ask
1. **What is the human intent?** Who is the user persona, and what is their entry point?
2. **What expectations define success or failure?** What must happen (happy path) and what must _never_ happen (safety boundaries, invariant constraints)?
3. **What evidence exists?** Are there existing specs, code behaviors, database schemas, or screenshots that anchor this feature?
4. **What events, commands, or contracts belong to this feature?**
   - Events (past tense: `x.created`, `y.approved`)
   - Commands (actions: `create.x`, `approve.y`)
   - Contracts (schemas: `x.config.v1`)
5. **What jobs, artifacts, verifiers, or approvals are needed?**
   - Jobs: Computations or background processors
   - Artifacts: Intermediate files, database records, compiled assets
   - Verifiers: Test suites, linters, risk engines, policy checkers
   - Approvals: Human-in-the-loop gates

### Stop criteria
- A valid local `dependency-map.yaml` exists, passes validation under `--strict`, and all expectations/verifiers have matching tests or check tasks defined.

### Agent must not guess
- The agent must not guess user configuration parameters, architectural policies, or security boundary definitions. These require explicit human confirmation.

### Commands to run
```bash
specdag validate specs/features/<feature-folder>/dependency-map.yaml --strict
```

---

## 2. Multi-Feature Integration Flow

### When to use
Use this when multiple features communicate across boundaries via shared events, contracts, or approvals.

### Required artifacts
- Decentralized local dependency maps for all participating features.
- Global event/contract catalogs under `specs/events/` or `specs/contracts/`.
- Assembled global map `dependency-map.global.json`.

### Questions to ask
1. **Which local feature maps are involved?** What are the feature paths?
2. **Which events or contracts connect them?** Who is the producer and who is the consumer?
3. **Are there node naming conflicts?** Do different features define the same node ID with conflicting titles or types?
4. **Are there global cycles?** Does feature A trigger feature B, which triggers feature C, which loops back to feature A?
5. **What is the downstream impact of a change?** If feature A's event structure changes, which downstream nodes in features B and C are affected?
6. **Which durable truth needs syncing?** Have shared events and contracts been moved into global catalogs?

### Stop criteria
- `specdag assemble` compiles all maps without cycles or naming conflicts, and `check-catalogs` validates all event/contract catalog references.

### Agent must not guess
- The agent must not assume contract ownership, auto-resolve naming conflicts, or override global cycles. These require human architectural decisions.

### Commands to run
```bash
specdag assemble specs/features/ -o specs/_generated/dependency-map.global.json
specdag check-catalogs specs/
```

---

## 3. Reconciliation Flow

### When to use
Use this when existing code, legacy specifications, runtime logs, or code knowledge graphs (e.g., GitNexus) disagree with the intended Spec-DAG.

### Required artifacts
- `dependency-map.yaml` (Normative Spec-DAG)
- Observed code graph (from GitNexus or similar tool) or execution traces
- `decisions.md` (Reconciliation log)

### Questions to ask
1. **What does the Spec-DAG intend?** Review the normative `dependency-map.yaml`. What is the desired behavior and obligation model?
2. **What does the code actually implement?** Analyze the observed codebase using code intelligence graphs (e.g., GitNexus) or AST call-graphs.
3. **What does the runtime execute?** Check logs, traces, or runtime events (`agent_run_trace`).
4. **Where are the mismatches?**
   - Missing implementation (Spec calls for a job, but no code exists).
   - Undeclared dependency (Code calls service B, but the Spec-DAG does not declare it).
   - Missing verifier (Expectation exists in spec, but no test/verifier exists in code).
5. **Is the code wrong, the spec wrong, or is a decision pending?**
6. **What evidence justifies the reconciliation?** Capture log payloads, AST paths, or benchmark results.

### Stop criteria
- Every mismatch has been recorded as a decision in `decisions.md` and both the spec/map and codebase have been updated to reflect the agreed state.

### Agent must not guess
- **Never vibe-code to resolve mismatches.** The agent must not silently rewrite the spec to match incorrect code, or refactor code to match an outdated spec. Record: `Evidence -> Implication -> Open Gap` first. Decide on the reconciliation path, document it, and then implement.

### Commands to run
```bash
specdag doctor specs/
specdag check-catalogs specs/
```
