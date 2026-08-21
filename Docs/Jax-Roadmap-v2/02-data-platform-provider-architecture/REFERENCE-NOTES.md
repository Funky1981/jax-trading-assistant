# Phase 02 Reference Notes — Data Platform

## Fincept concepts worth extracting
- `src/datahub/DataHub.h`: topic ownership, subscription, TTL/freshness, errors, last-known-good, request coalescing and rate limiting.
- `src/services/data_normalization/DataNormalizationService.h`: raw + normalized data, mapping identity, transforms, schema validation and persistence.
- `src/mcp/tools/DataConnectorManifest.inc`: large connector inventory useful as a discovery catalogue.
- `src/network/`: HTTP/WebSocket adapter boundaries.

## Important caveat
The current connector manifest explicitly describes many connectors as not invoked by a real source callsite. Treat connector existence as **candidate discovery**, not production proof.

## Jax-specific design target
A provider must terminate in a stable canonical contract with:
- raw immutable payload/reference,
- observed/received/normalized timestamps,
- source/provider identity,
- schema/version,
- freshness and quality,
- validation errors,
- retry/rate-limit/health state.

Provider details must not leak into recommendation logic.
