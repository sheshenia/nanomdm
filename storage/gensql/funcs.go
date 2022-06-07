package gensql

import (
	"database/sql"
	"errors"
)

var (
	ErrNoCert               = errors.New("no certificate in MDM Request")
	ErrEmptyDriverName      = errors.New("empty driver name")
	ErrUnsupportedSQLDriver = errors.New("unsupported SQL driver")
)

// NullEmptyString returns a NULL string if s is empty.
func NullEmptyString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}
