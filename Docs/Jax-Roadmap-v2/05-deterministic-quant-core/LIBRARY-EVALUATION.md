# Phase 05 library evaluation

**Decision:** retain the bounded Go implementation for Phase 05; add no
third-party quant or Python runtime dependency.

## Evaluation evidence

| Candidate | Repository/runtime observation | Phase-05 decision |
| --- | --- | --- |
| NumPy | Importable in the host Python environment, but absent from Jax's declared service dependencies | Do not add; the phase needs small scalar/vector formulas and a new runtime would add deployment/version surface without improving the contract |
| SciPy | Not importable in the host environment and not declared by Jax | Do not add; no Phase-05 calculation requires its advanced numerical routines |
| statsmodels | Not importable in the host environment and not declared by Jax | Do not add; regression beyond deterministic beta is outside this phase |
| skfolio | Not importable in the host environment and not declared by Jax | Do not add; portfolio optimization/model selection is later-scope capability |
| Riskfolio-Lib | Not importable in the host environment and not declared by Jax | Do not add; advanced portfolio optimization and tail-risk workflows are later-scope capability |

The reproducible checks were:

```text
python -c "import importlib.util; names=['numpy','scipy','statsmodels','skfolio','riskfolio']; print({name: bool(importlib.util.find_spec(name)) for name in names})"
{'numpy': True, 'scipy': False, 'statsmodels': False, 'skfolio': False, 'riskfolio': False}

go list -m all | Select-String -Pattern 'gonum|stats|portfolio|numpy|scipy'
<no matching declared module>
```

The Go standard library provides the required `math`, canonical JSON, hashing,
sorting, and validation primitives. All Phase-05 outputs remain behind the
versioned request/response boundary, with explicit algorithm versions and
frozen-input identity. A later optimization or advanced statistical phase may
re-evaluate a library when its capability, pinned version, licensing, runtime
boundary, and independent numerical oracle are specified.
