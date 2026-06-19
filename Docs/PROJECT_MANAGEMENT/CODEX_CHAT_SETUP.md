# Codex Chat Setup for ProjectOS

Use this file when working with Codex in ChatGPT for Jax or any other personal project.

## Important Separation

Codex Code is for work only.

Codex is used for:

- Jax
- Personal GitHub projects
- Home server projects
- OSINT/research platform
- News app
- Exercise microservice
- Other non-work projects

## First Prompt for an Existing Project

Paste this into Codex when adding ProjectOS to an existing repo:

```text
Read /ai/commands/baseline-existing-project.md and follow it.

Inspect this repo and create a current-state baseline. Do not guess. Mark unknowns clearly.

Update the /project files so future work has:
- a project brief
- a roadmap
- a backlog
- current focus
- a decision log
- a risk log
- a release log

Use /ai/AGENTS.md as the operating rules.
Do not write feature code during this baseline step.
```

## First Prompt for a New Project

```text
Use ProjectOS for this project.

Start by helping me complete:
- /project/brief.md
- /project/roadmap.md
- /project/backlog.md
- /project/current-focus.md

Do not write code yet. First create the project plan, phases, risks, and first 3 actions.
```

## Feature Planning Prompt

```text
Read /ai/commands/plan-feature.md and follow it.

Feature: [describe feature]

Do not write code yet.

Link this feature to:
- the roadmap
- a backlog item
- acceptance criteria
- likely files to change
- risks
- review steps
```

## Implementation Prompt

```text
Read /ai/commands/implement-task.md and follow it.

Implement only the approved planned task.

Keep the change focused.
Update ProjectOS files after the implementation.
Summarise files changed, tests/checks, ProjectOS updates, and risks/follow-ups.
```

## Review Prompt

```text
Read /ai/commands/review-task.md and follow it.

Review the current changes against:
- the roadmap
- the backlog item
- acceptance criteria
- tests/checks
- architecture impact
- risk impact
- ProjectOS updates

Be strict. Separate blockers from optional improvements.
```

## Weekly Project Review Prompt

```text
Read the ProjectOS files and give me:

1. Current project state
2. What changed recently
3. What is blocked
4. Top risks
5. What should be done next
6. Whether the roadmap still makes sense
7. Any ProjectOS files that are stale
```

## Non-Negotiable Rule

No coding starts until Codex has checked the roadmap, backlog, current focus, decisions, and risks.
