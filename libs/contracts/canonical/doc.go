// Package canonical owns Jax's versioned cross-runtime domain vocabulary.
//
// Types in this package are domain records, not database rows, HTTP DTOs,
// provider payloads, broker messages, or AI response shapes. Boundary code may
// adapt those representations into these contracts, but those representations
// do not define canonical meaning.
//
// Contract values use record semantics: callers construct a complete value,
// validate it, and replace it with a newly identified/versioned record when its
// historical meaning changes. Persistence and append-only audit policy are
// deliberately outside this package.
package canonical
