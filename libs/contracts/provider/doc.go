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
// accepted results. It does not own provider DTOs or fetch provider data.
package provider
