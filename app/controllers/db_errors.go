package controllers

import "strings"

// isUniqueViolation reports whether an error is a unique-constraint violation.
//
// The check is a string match on purpose: importing the SQLite and PostgreSQL
// driver error types would add dependencies and couple this file to each
// backend. The result only maps the HTTP status (a race that loses the unique
// index check becomes a 409 instead of a 500), never a security decision.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") ||
		strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate")
}
