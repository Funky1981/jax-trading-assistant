// Package provider owns Jax's versioned provider registry and capability
// vocabulary.
//
// Provider definitions describe stable provider identity and static capability
// support. They do not contain endpoints, credentials, machine-local settings,
// normalized records, health probes, retry policy, or production storage.
// Provider boundary representations remain explicitly raw and provider-owned;
// immutable raw payload references bind exact received bytes to acquisition,
// capability, schema, retention, and abstract storage semantics. Canonical
// outputs remain declared destination schemas from package canonical and are
// never produced by this package.
package provider
