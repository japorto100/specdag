# Intent and Expectations Template

Use this for Level 1+ feature specs. Keep intent human-owned and expectations
checkable. Context belongs in evidence sections, not inside the intent.

## Intent

```markdown
## Intent

### Wanted Outcome

<What the user/business/system owner actually wants. No stack choices unless
they are part of the requirement.>

### Constraints

| Constraint | Why it matters |
|---|---|
| ... | ... |

### Success Scenarios

- <Observable scenario where the outcome is correct.>

### Failure Scenarios

- <Observable scenario where the outcome is wrong, unsafe, incomplete, or
  misleading.>

### Connected Intents / Features

- <Feature or domain link>: <why it is connected>
```

## Expectations

```markdown
## Expectations

### Done Means

- <User/domain-visible condition that proves success.>

### Failed Means

- <Condition that must be treated as failure, regression, or blocked work.>

### Must Not Happen

- <Boundary the implementation must stay inside.>

### Agent-Verifiable Checks

| Check | Evidence |
|---|---|
| <build/test/smoke/contract/manual check> | <expected output or artifact> |
```

## Agent Guessing Gaps

```markdown
## Agent Guessing Gaps

| Gap | Why the agent must not guess | Resolution |
|---|---|---|
| ... | Product/security/data/architecture consequence. | decided/deferred/open |
```

Close gaps before implementation when they affect product behavior, data
ownership, security posture, migrations, money, privacy, irreversible
architecture, or public contracts.

