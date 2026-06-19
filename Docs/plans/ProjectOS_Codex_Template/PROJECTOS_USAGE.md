# ProjectOS Usage Guide

## New Project Workflow

1. Copy ProjectOS into the repo.
2. Complete `/project/brief.md`.
3. Complete `/project/roadmap.md`.
4. Create initial backlog items in `/project/backlog.md`.
5. Set `/project/current-focus.md`.
6. Create GitHub Project board.
7. Create GitHub issues for first tasks.
8. Start coding only from planned issues.

## Existing Project Workflow

1. Copy ProjectOS into the repo.
2. Ask Codex to run:
   `/ai/commands/baseline-existing-project.md`
3. Review the generated baseline.
4. Fix anything inaccurate.
5. Create GitHub issues from backlog.
6. Continue all work through the ProjectOS workflow.

## Codex Prompt for Existing Project

```text
Read /ai/commands/baseline-existing-project.md and follow it.

Inspect this repo and create a current-state baseline. Do not guess. Mark unknowns clearly. Update the /project files so future work has a roadmap, backlog, decision log, risk log, release log, and current focus.
```

## Codex Prompt for New Feature

```text
Read /ai/commands/plan-feature.md and follow it.

Feature: [describe feature]

Do not write code yet. Link this to the roadmap and backlog. Produce a plan with acceptance criteria, likely files to change, risks, and review steps.
```

## Codex Prompt for Implementation

```text
Read /ai/commands/implement-task.md and follow it.

Implement the planned task only. Keep the change focused. Update ProjectOS files after the implementation.
```

## Codex Prompt for Review

```text
Read /ai/commands/review-task.md and follow it.

Review the current changes against the task acceptance criteria and ProjectOS rules. Be strict. Separate blockers from optional improvements.
```

## Weekly Project Review

Once per week, ask:

```text
Read the ProjectOS files and give me:
1. Current project state
2. What changed this week
3. What is blocked
4. Top risks
5. What should be done next
6. Whether the roadmap still makes sense
```

## Minimum Discipline

No task is done unless:

- Backlog status updated
- Current focus updated
- Decisions updated if needed
- Risks updated if needed
- Release notes updated if needed
