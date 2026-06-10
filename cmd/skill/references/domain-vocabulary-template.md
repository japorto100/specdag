# Domain Vocabulary Template

Each bounded context should have a `vocabulary.md` in `domains/<context>/` once
the context has more than a handful of important terms. This prevents drift
where the same concept gets different names in specs, code, and conversations.

## Template

```markdown
# <Domain> Vocabulary

## Core Terms

| Term | Definition | Synonyms (allowed) | Anti-patterns (avoid) |
|---|---|---|---|
| ... | One sentence. | Acceptable aliases. | Terms that look similar but mean something else. |

## Commands

| Command | Meaning | Issued by |
|---|---|---|
| ... | What it asks the system to do. | Actor or system. |

## Queries

| Query | Returns | Used by |
|---|---|---|
| ... | What data shape. | Consumer. |

## Policies

| Policy | Rule | Enforced by |
|---|---|---|
| ... | Business invariant in plain language. | Module or service. |

## Ownership

| Aspect | Owner |
|---|---|
| Vocabulary authority | <team-or-person> |
| Schema evolution | <team-or-person> |
```

## Guidance

The anti-patterns column matters. Terms like `status`, `state`, `event`, and
`session` often mean several different things in one codebase. If the domain
uses one meaning, name the meanings that are explicitly not intended.

Keep vocabularies concise. If a term needs a paragraph, move that explanation to
`domains/<context>/policies.md`, a contract, or a design decision.

