// Package provider owns Jax's versioned provider registry and capability
// vocabulary.
//
// Provider definitions describe stable provider identity and static capability
// support. They do not contain endpoints, credentials, machine-local settings,
// raw payloads, normalized records, health probes, retry policy, or storage.
// Provider boundary representations remain explicitly raw and provider-owned;
// canonical outputs are only declared destination schemas from package
// canonical.
package provider
