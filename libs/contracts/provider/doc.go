// Package provider owns Jax's versioned provider registry and capability
// vocabulary.
//
// Provider definitions describe stable provider identity and static capability
// support. They do not contain endpoints, credentials, machine-local settings,
// normalized records, health probes, retry policy, or production storage.
// Provider boundary representations remain explicitly raw and provider-owned;
// immutable raw payload references bind exact received bytes to acquisition,
// capability, schema, retention, and abstract storage semantics. Canonical
// outputs remain owned and validated by package canonical. This package also
// owns the deterministic normalization boundary that verifies exact raw bytes,
// routes by provider/capability/raw schema, invokes provider-owned parsers and
// mappings, validates canonical output and provenance, and returns only fully
// accepted results. Versioned provider-neutral freshness policies then
// evaluate exact normalized records at caller-supplied times, and explicit
// last-known-good policy can deterministically select a same-key immutable
// record without upgrading stale data. A bounded operational executor then
// applies explicit operation safety, failure classification, retry/backoff,
// Retry-After, process-local rate-limit state, capability health assessment,
// and typed instrumentation through injected time. Successful acquisition
// returns exact bytes for the raw-persistence boundary; it never normalizes or
// silently returns fallback data. Data freshness, normalization quality,
// fallback use, and provider health remain separate dimensions. The package
// does not own provider DTOs, production HTTP clients, provider quotas, health
// persistence, or freshness/LKG state.
package provider
