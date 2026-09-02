// Package sec implements the bounded official SEC/EDGAR evidence adapter.
//
// The package owns SEC wire parsing and deterministic normalization only. Its
// public results contain canonical Jax Evidence/Observation records plus the
// source semantics needed to interpret a filing or XBRL fact; SEC JSON response
// shapes do not cross the adapter boundary.
package sec
