# Risk Log

Use this file to track technical, delivery, product, security, cost, and operational risks.

## Risk Rating

Likelihood:

- Low
- Medium
- High

Impact:

- Low
- Medium
- High

## Template

### RISK-000 — Risk Title

**Status:** Open / Mitigating / Accepted / Closed  
**Likelihood:** Low / Medium / High  
**Impact:** Low / Medium / High  
**Owner:** TBC  

#### Description

What could go wrong?

#### Trigger

What would indicate this risk is happening?

#### Mitigation

How will this be reduced?

#### Contingency

What is the fallback plan?

---

## Risks

### RISK-001 — Project documentation becomes stale

**Status:** Open  
**Likelihood:** Medium  
**Impact:** High  
**Owner:** TBC  

#### Description

ProjectOS files may become inaccurate if implementation work is done without updating docs.

#### Trigger

Code changes are merged without updates to roadmap, backlog, decisions, risks, or releases.

#### Mitigation

Use pull request checklist and AI review commands before merge.

#### Contingency

Run `/ai/commands/baseline-existing-project.md` again to refresh the project state.
