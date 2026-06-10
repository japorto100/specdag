# Fullstack Event Layout Example

Use this when a repo has many domains, runtimes, and agents. It is generic and
not tied to a specific stack.

## Repository Layout

```text
specs/
  foundations/
    001-platform-direction/
      spec.md
      feature_map.md
      open_questions.md
      decisions.md
      evidence/

  features/
    002-agent-workbench/
      spec.md
      event-flow.md
      contracts/
        stream.md
        api.md
      tasks.md
      decisions.md
      evidence/

  domains/
    agent/
      session-model.md
      context-policy.md
      vocabulary.md
    collaboration/
      rooms-and-spaces.md
      share-policy.md
    trading/
      symbols-signals-portfolios.md
      risk-policy.md
    auth/
      identity-and-devices.md

  events/
    catalog.md
    flows/
      agent-run.md
      publish-artifact.md
    schemas/
      agent.run.started.schema.json
      artifact.published.schema.json

  contracts/
    api/
      public-rest.md
    auth/
      session-sync.md
    deep-links/
      app-links.md
    sync/
      offline-replay.md

  archive/
    2026-01-10-agent-spike/
      disposition.md
```

## Vertical Feature Rule

A feature owns one user-visible outcome even when it touches several systems.
Do not split it into separate feature specs for backend, web, mobile, and
workers. Put runtime-specific tasks in `tasks.md`.

```text
features/002-agent-workbench/tasks.md
  Backend
  Web
  Mobile
  Event contracts
  Verification
```

## Horizontal Knowledge Rule

Use horizontal folders only for knowledge that must outlive one feature:

- `domains/`: vocabulary, policies, ownership, bounded contexts.
- `events/`: catalog, schemas, producer/consumer maps, flows.
- `contracts/`: APIs, auth, links, protocol, compatibility.
- `foundations/`: architecture direction and multi-feature decisions.

## Event Flow Template

```text
# Event Flow: <feature>

Status: draft | accepted | superseded
Owner:
Related feature:

## Flow

1. User or system action:
2. Command/query, if any:
3. Event emitted:
4. Consumers:
5. User-visible state change:

## Events

### domain.event.name

Meaning:
Producer:
Consumers:
Schema:
Causation ID:
Correlation ID:
Ordering:
Idempotency:
Replay:
Failure behavior:
Privacy class:
Compatibility:
```
