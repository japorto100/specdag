# Event Catalog Template

Use this only when a project has enough durable domain or integration events
that scattered prose becomes hard to maintain.

Optional future tool: EventCatalog (https://www.eventcatalog.dev/) can render and
query domains, services, messages, schemas, owners, and flows from Git-versioned
architecture docs. This template is intentionally close to those primitives, but
EventCatalog is not required for normal feature work.

Do not catalog every internal function call, UI state change, local callback, or
one-off implementation step. If an event is not published, replayed, stored, or
consumed across a meaningful boundary, keep it in the feature `event-flow.md`.

## Layout

```text
events/
  catalog.md
  <domain>/
    <entity>.<past-tense-verb>.md
  schemas/
    <event-name>.schema.json
```

Keep `catalog.md` as a one-line-per-event index that links to individual files.

## Entry Template

```markdown
---
name: <domain>.<entity>.<past-tense-verb>    # e.g. order.payment.captured
version: "1.0.0"                             # semver; bump on breaking schema change
status: draft | accepted | deprecated
domain: <bounded-context-name>
producer: <service-or-module>
consumers:
  - <consumer-1> (<why>)
  - <consumer-2> (<why>)
delivery: synchronous | at-most-once | at-least-once | exactly-once
idempotent: true | false
ordering: none | per-key | total
replay_safe: true | false
privacy_class: public | internal | pii | sensitive
tenant_scoped: true | false
schema_ref: <relative-path-or-url>           # optional: JSON Schema / Protobuf / Avro
---

# <event-name>

## Why this event exists

<One paragraph: business reason, not technical plumbing.>

## Schema

<Field table or inline schema. Keep it readable; link to full JSON Schema
via schema_ref for machine consumption.>

| Field | Type | Required | Description |
|---|---|---|---|
| ... | ... | ... | ... |

## Invariants

<Conditions that MUST hold when this event is emitted. Violations are bugs.>

## Compatibility

<Backward/forward rules. What can consumers assume stays stable?>
```
