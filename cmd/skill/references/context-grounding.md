# Context Grounding

Use this when a feature touches an existing codebase, multiple runtimes, auth,
data, agents, or external packages.

## Principle

Context should be progressively loaded, sourced, and checkable. A large prompt or
chat history is not reliable context by itself.

## Context Brief Template

```markdown
## Context / Evidence

| Evidence | Source | Implication | Open Gap |
|---|---|---|---|
| ... | file path / command / source URL / date | ... | ... |
```

## Recommended Sources

- Local instructions: `AGENTS.md`, `CLAUDE.md`, repo README, ADRs.
- Existing specs/features/domains/contracts/events.
- Source files, tests, package manifests, config files, migrations, CI.
- Runtime evidence: command output, screenshots, logs, payload samples.
- External facts from official or primary sources when packages, standards,
  security, laws, prices, or current behavior may have changed.

## Rules

- Prefer source paths and command output over remembered architecture.
- Mark stale docs as risk instead of treating them as truth.
- Do not overload feature specs with every file read. Curate only evidence that
  changes a decision or constrains implementation.
- If the agent would need to guess, record a gap rather than silently deciding.
- Refresh context after large interruptions, branch changes, rebase, or generated
  code bursts.

