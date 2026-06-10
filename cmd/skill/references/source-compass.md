---
title: Source Compass - Intent-Event-Contract Development
status: reference
updated: 2026-05-28
---

# Source Compass

This file is a compact map of sources and adjacent methods that inform the
skill. It is not a project-specific research file. Put product/library-specific
sources in the relevant feature's `research.md`.

## Use As Method Input

### Intent-driven critique of SDD

- Source: Kapil Viren Ahuja, "The Method That Replaces Spec-Driven Development
  - IDSD", Medium, 2026-05-20.
- Link:
  https://medium.com/activated-thinker/the-method-that-replaces-spec-driven-development-idsd-66e921f6cdf7
- Use for: the separation of human-owned Intent, Expectations and progressively
  gathered Context.
- Do not treat as: an implementation framework, primary engineering standard,
  or project-specific authority.

### Anatomy of Intent / ICE follow-up

- Source: Kapil Viren Ahuja, "The Anatomy of Intent: ICE in IDSD..."
- Link:
  https://medium.com/activated-thinker/the-anatomy-of-intent-ice-in-idsd-built-from-where-spec-driven-breaks-1597e5a16659
- Use for: refining the five parts of intent: wanted outcome, constraints,
  success scenarios, failure scenarios and connections.

### OpenSpec

- Source: Fission-AI OpenSpec.
- Link: https://github.com/Fission-AI/OpenSpec
- Use for: lightweight change folders, proposal/tasks/spec/archive workflow,
  and the "agree before build" pattern.
- Do not copy blindly: this skill keeps Intent and Expectations explicit and
  separates Evidence, Events and Contracts more strongly for fullstack systems.

### Symphony / agent harness design

- Source: OpenAI Symphony announcement and spec.
- Link:
  https://openai.com/index/open-source-codex-orchestration-symphony/
- Use for: agent-harness thinking, proof-of-work artifacts, automation control
  planes and validating specs through implementations.
- Do not treat as: the default structure for ordinary feature specs.

### EventCatalog

- Source: EventCatalog.
- Link: https://www.eventcatalog.dev/
- Use for: future catalog direction when domains, services, events, commands,
  schemas, owners and flows become numerous enough to need first-class docs.
- Do not install by default. Keep current markdown artifacts structured so they
  can be mapped to EventCatalog later.

## Source Hygiene

- Prefer official docs, standards, repository docs and primary sources for
  technical/library claims.
- Use opinion articles as framing only; record them as influence, not authority.
- Keep source notes short. Long summaries belong in feature research or archive
  evidence, not in the skill body.
- Re-check fast-moving tool claims before implementation.
