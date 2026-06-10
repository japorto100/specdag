---
name: event-spec-driven-development
description: Use when designing or changing agentic, fullstack, event-driven, or contract-sensitive features with intent, acceptance expectations, context grounding, message flows, APIs, streams, specs, plans, tasks, research capture, or archive/disposition.
---

# Intent-Event-Contract-Driven Development

## Overview

Use human-owned intent and expectations before implementation commitments, then
ground the agent in evidence and make events and contracts the durable system
language. Specs, plans, and tasks are working containers around those contracts,
not the source of truth by themselves.

Core rule:

```text
Intent first.
Expectations second.
Evidence and context third.
Bounded contexts fourth.
Events and contracts fifth.
Implementation tasks sixth.
Verification and sync always.
```

## Severity

Pick the lightest level that protects the change.

| Level | Use for | Required artifacts |
|---|---|---|
| 0 Small Patch | isolated code/doc tweak | no feature spec; normal tests/checks |
| 1 Focused Feature | user-visible change in one area | `spec.md` with intent, expectations, and guessing gaps; `tasks.md`; `plan.md` if useful |
| 2 Evented/Cross-System Feature | APIs, streams, auth, data sync, UI state, agents, or multiple runtimes | Level 1 + `event-flow.md`, `contracts/`, `decisions.md`, `evidence/`, `research.md`, `dependency-map.yaml` |
| 3 Foundation/Epic | product direction, architecture, many related features or domains | Level 2 + `feature_map.md`, `open_questions.md`, domain/event/catalog structure, archive rules |

If unsure between two levels, choose the higher level until discovery proves the
scope is smaller.

## Organization

For large fullstack systems, do not create separate specs per programming
language or runtime. Organize around vertical features and horizontal durable
knowledge:

```text
specs/
  foundations/   # platform direction and cross-cutting architecture
  features/      # user-visible vertical slices
  domains/       # bounded contexts, vocabulary, policies, ownership
  events/        # event catalog, schemas, flows, producer/consumer map
  contracts/     # APIs, auth, deep links, sync, compatibility rules
  archive/       # spikes and superseded work with disposition
```

Use `features/NNN-name/` for work that spans mobile, web, backend, agents, and
protocols. Put language-specific work inside `tasks.md`, not in separate specs.

For an example layout, read `references/fullstack-event-layout.md`.

## Workflow

1. **Intent**
   - State the human-owned outcome, constraints, success scenarios, failure
     scenarios, and connected intents/features.
   - Keep intent free of implementation choices unless the choice is itself a
     product or compliance constraint.

2. **Expectations**
   - Define what counts as done, what counts as failed, what must not happen,
     and which checks can prove it.
   - Write expectations in user/domain language first, then map them to tests,
     builds, screenshots, payloads, or contract checks.

3. **Discovery and context grounding**
   - Read local instructions, README/ADRs/specs, relevant source files,
     manifests, package/tool configs, tests, and CI.
   - For external/fast-moving tech, verify with official or primary sources.
   - Capture findings as `Evidence -> Implication -> Open Gap`.
   - Record any `Agent Guessing Gaps`: decisions the agent must not invent.

4. **Bounded contexts**
   - Name the domain that owns each fact, event, command, query, and policy.
   - Do not let generic UI or transport names hide domain meaning.
   - Separate internal implementation events from published integration events.
   - Create a `vocabulary.md` per bounded context in `domains/<context>/` once
     the domain has more than a handful of terms. Use
     `references/domain-vocabulary-template.md` when needed.

5. **Events and contracts**
   - Define producers, consumers, payload schemas, owners, compatibility,
     causation/correlation IDs, ordering, replay/idempotency, privacy, and
     failure behavior.
   - Use past-tense names for events. Model commands and queries separately.
   - Treat event names as draft until owning domains and consumers are known.

6. **Feature behavior**
   - Define user-visible goal, non-goals, requirements, acceptance criteria,
     risks, and open questions.
   - Keep product decisions separate from implementation details.
   - Use precise status labels: `draft`, `accepted`, `superseded`, `archived`,
     `spike`, `deferred`.

7. **Plan** - map behavior to modules/files/contracts, name dependency and
   migration risks, and select verification gates by risk.

8. **Tasks** - create small checkable tasks, including discovery and
   validation. Mark completed work only when evidence exists.

9. **Implementation** - do not let a spike become product direction by inertia.
   Keep contracts explicit and update specs when reality changes.

10. **Verification and sync** - run selected checks and record important
    outputs. If validation fails, update the task/plan loop before claiming
    done. Move durable decisions into `domains/`, `contracts/`, `events/`, or
    `foundations/` when the feature stops being the source of truth.

## Artifact Layout

Recommended feature layout:

```text
specs/
  features/
    001-feature-name/
      spec.md             # intent, expectations, context, gaps
      event-flow.md       # Level 2+
      dependency-map.yaml # Level 2+
      plan.md             # if needed
      tasks.md
      research.md         # curated findings
      decisions.md        # real decisions
      contracts/          # APIs, event schemas, links, policies
      evidence/           # screenshots, logs, outputs, payloads
      research/raw/       # only decision-relevant raw material
  archive/
    YYYY-MM-DD-name/
      spec.md
      plan.md
      tasks.md
      disposition.md
```

Minimal transitional layout is acceptable in existing repos:

```text
specs/
  001-feature-name/
    spec.md
    event-flow.md
    plan.md
    tasks.md
    contracts/
    evidence/
```

For a review checklist, read `references/artifact-checklist.md`.
For intent and expectations templates, read
`references/intent-expectations-template.md`.
For context-grounding guidance, read `references/context-grounding.md`.
For the method source compass and links to adjacent approaches, read
`references/source-compass.md`.

## Intent and Expectations Rules

- Intent belongs to the human requesting or owning the outcome.
- Expectations define the boundary of done and failed; they are not a place to
  smuggle implementation guesses.
- Context is discovered progressively from the repo, tools, primary sources, and
  runtime evidence. Do not dump unrelated context into the spec.
- Every meaningful feature must list `Agent Guessing Gaps`: places where an
  agent would otherwise decide product behavior, ownership, security posture,
  data semantics, or irreversible architecture.
- If intent or expectations change during implementation, update the spec before
  continuing. Do not let the diff become the new intent by accident.

## Event Contract Rules

- Distinguish occurrence, event, message, topic/stream, command, query, policy,
  and status snapshot. Do not use "event" as a catch-all word.
- Name owner, producer, consumers, and transport separately.
- Record why the event exists, not just what fields it has.
- Include schema versioning and backward/forward compatibility expectations.
- Capture causation ID, correlation ID, actor, timestamp source, and tenant or
  workspace scope when relevant.
- State delivery semantics: at-most-once, at-least-once, retry, replay,
  duplicate handling, ordering, and poison-message behavior.
- Mark privacy/security class and redaction rules before implementation.
- Prefer an event catalog over scattered prose when events become numerous.

When the project has more than ~10 durable domain or integration events,
maintain one file per event inside `events/<domain>/`. Use YAML frontmatter for
machine-readable metadata and markdown body for human context. This keeps the
catalog grep-friendly and diff-friendly across branches.

Do not create catalog entries for every internal function call, UI state change,
or one-off implementation flow. If an event is not published, replayed, stored,
or consumed across a meaningful seam, keep it in `event-flow.md` instead of
promoting it into the catalog.

Keep `catalog.md` in `events/` as a one-line-per-event index that links to the
individual files. Regenerate it or keep it by hand — either works as long as it
stays current.

Optional future tool: EventCatalog (https://www.eventcatalog.dev/) can document,
visualize, and query domains, services, messages, schemas, owners, and flows from
Git-versioned architecture docs. Do not require it by default, but keep event and
domain artifacts structured enough that they can be mapped to EventCatalog later.

For a catalog entry template, read `references/event-catalog-template.md`.
For bounded-context vocabulary guidance, read
`references/domain-vocabulary-template.md`.

## Evidence Rules

- Curate findings in `research.md`; raw dumps are not a substitute.
- Keep context provenance: file paths, command outputs, source links, dates, and
  what conclusion each piece of evidence supports.
- Store raw chats/searches only when decision-relevant.
- Decisions need date, status, evidence, rejected alternatives, and follow-up.
- Contracts must distinguish draft from final.
- Evidence should be durable: command output, screenshots, payload samples,
  benchmarks, build/test logs.

## Archive Rules

Spikes are evidence, not automatically product code. When archiving or
superseding a spike/spec, add `disposition.md` with status, replacement link,
what shipped/was learned, reusable evidence, known gaps, unfinished work, where
future work moved, and linked commits/artifacts when available.

Do not delete or silently renumber active evidence. Preserve provenance.

## Sync Rules

Feature folders are working containers. After implementation, move durable truth
to the horizontal layer it belongs to:

- `domains/`: vocabulary, ownership, policies, bounded contexts.
- `contracts/`: APIs, auth/session rules, data contracts, compatibility.
- `events/`: accepted published events, schemas, flows, producer/consumer maps.
- `foundations/`: cross-feature architecture decisions and platform direction.

If a feature is closed without durable sync, record why in `decisions.md` or
`archive/*/disposition.md`.

## Verification Selection

Choose checks by affected surface:

- **Everyday:** typecheck/build, focused tests, format/lint touched surfaces.
- **CI parity:** repo CI-equivalent commands with warnings/settings.
- **High risk:** audits, policy scans, contract tests, security/privacy review,
  performance/size checks.
- **UI:** smoke run plus screenshot/manual evidence for important states.
- **Cross-system:** contract tests, sample payloads, compatibility and
  failure-mode evidence.

Never claim completion without saying which gates ran and which did not.

## Common Mistakes

- Letting the agent infer intent or definition of done from a broad prompt.
- Writing specs from memory without reading code.
- Mixing intent, expectations, and context into one vague wall of text.
- Treating Spec Kit style as a mandatory ceremony for every change.
- Creating separate feature specs per language instead of one vertical feature.
- Hiding business meaning inside transport topics, UI component names, or HTTP
  endpoint names.
- Freezing event/API names before the owning systems are stable.
- Calling commands or status snapshots "events" without modeling intent.
- Treating chat history as evidence without curating decisions.
- Letting generated spike code become architecture by accident.
- Running only the cheapest check when the change touches auth, crypto,
  persistence, money, privacy, or external contracts.

## Dependency Map Rules

For Level 2+ features, maintain a lightweight dependency map (`dependency-map.yaml`) when the feature involves multiple events, contracts, jobs, verification gates, or human approvals.

- **Graph types:**
  - Use DAGs (Directed Acyclic Graphs) for workflows, task execution, artifact derivation, and verification chains.
  - Do not force cyclic domain knowledge or feedback loops into a DAG. Keep them in domain models.
- **Allowed Nodes:**
  - `intent`, `expectation`, `event`, `command`, `query`, `contract`, `job`, `artifact`, `verifier`, `approval`
- **Allowed Edges:**
  - `defines_success_for` (intent -> expectation)
  - `triggers` (event -> job/command)
  - `produces` (job -> artifact/event)
  - `consumes` (job/command -> artifact/event)
  - `verified_by` (artifact -> verifier)
  - `verifies` (verifier -> expectation)
  - `requires_approval` (job/event -> approval)
- **Validation:**
  - Every node ID must be unique.
  - No cycles in DAG-type workflows.
  - All referenced events/contracts must exist in the repository's catalogs.

## Tooling

Use the `specdag` CLI tool or MCP server to manage and validate dependency maps:
- **Validate local map:** `specdag validate specs/features/NNN-name/dependency-map.yaml`
- **Assemble global map:** `specdag assemble specs/features/` (scans all features and builds a unified map, checking for global conflicts/cycles)
- **Render Mermaid diagram:** `specdag render specs/features/NNN-name/dependency-map.yaml`
- **Start MCP Server:** `specdag mcp` (exposes validation, assembly, and rendering tools to AI agents over standard I/O)
