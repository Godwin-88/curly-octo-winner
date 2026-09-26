package httputil

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres SQLSTATE codes we can translate into something a user can act on.
const (
	sqlStateUniqueViolation     = "23505"
	sqlStateForeignKeyViolation = "23503"
	sqlStateCheckViolation      = "23514"
)

// pgError extracts the underlying *pgconn.PgError from an error chain, if any.
func pgError(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}
	return nil
}

// IsUniqueViolation reports whether err is a unique constraint violation.
//
// Curriculum codes, contact phone numbers and fee codes are all unique per
// tenant, so this is the difference between "that KICD code is already used by
// Environmental Studies" and a raw "duplicate key value violates unique
// constraint" leaking to the browser.
func IsUniqueViolation(err error) bool {
	pgErr := pgError(err)
	return pgErr != nil && pgErr.Code == sqlStateUniqueViolation
}

// IsForeignKeyViolation reports whether err is a foreign key violation.
func IsForeignKeyViolation(err error) bool {
	pgErr := pgError(err)
	return pgErr != nil && pgErr.Code == sqlStateForeignKeyViolation
}

// IsCheckViolation reports whether err is a CHECK constraint violation.
func IsCheckViolation(err error) bool {
	pgErr := pgError(err)
	return pgErr != nil && pgErr.Code == sqlStateCheckViolation
}

// RespondWriteError maps a failed write onto the right status code.
//
// Handlers call this after any insert/update/delete that a user triggered.
// `conflict` is the message to show when the write collided with existing data
// (for example a KICD code already in use); `badRef` covers a reference to a
// row that does not exist in this tenant.
//
// A constraint the API did not anticipate still falls through to 500, because
// a genuine bug must never masquerade as a validation message.
func RespondWriteError(w http.ResponseWriter, err error, conflict, badRef string) {
	switch {
	case IsUniqueViolation(err):
		RespondConflict(w, "ALREADY_EXISTS", conflict)
	case IsForeignKeyViolation(err):
		RespondBadRequest(w, "INVALID_REFERENCE", badRef)
	default:
		RespondInternalError(w, err)
	}
}
