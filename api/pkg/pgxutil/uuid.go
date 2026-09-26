// Package pgxutil holds small helpers for passing Go values to pgx v5.
package pgxutil

import "github.com/google/uuid"

// UUIDArray converts a []uuid.UUID into the []string form pgx v5 can encode as
// a text[] parameter (Postgres casts it to uuid[] with an explicit ::uuid[]).
//
// pgx v5 has no codec registered for []uuid.UUID, so passing that type as a
// query argument fails at execution time with:
//
//	cannot use unregistered type []uuid.UUID as query argument in QueryExecModeExec
//
// This silently broke every write that passes a guardian id list (learner
// create/update, custom SMS/WhatsApp audiences), so all call sites go through
// here. A nil or empty slice returns an empty (non-nil) []string, which pgx
// encodes as an empty array rather than NULL — the columns are NOT NULL with a
// default, so an empty array is the correct value.
func UUIDArray(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
