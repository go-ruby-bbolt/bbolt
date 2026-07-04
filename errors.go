package bbolt

import (
	"errors"

	bolt "go.etcd.io/bbolt"
)

// Error is a Bolt error. It names the Ruby exception class a host binding should
// raise (Class) and wraps the underlying bbolt sentinel, so both
//
//	errors.Is(err, bbolt.ErrBucketNotFound)      // this package's sentinel
//	errors.Is(err, bolt.ErrBucketNotFound)       // go.etcd.io/bbolt's sentinel
//
// report true.
type Error struct {
	// Class is the Ruby exception class the host should raise, e.g.
	// "Bolt::BucketNotFound".
	Class string
	err   error // the wrapped go.etcd.io/bbolt sentinel
}

// Error implements the error interface, reporting the underlying bbolt message.
func (e *Error) Error() string { return e.err.Error() }

// Unwrap exposes the underlying go.etcd.io/bbolt sentinel so errors.Is/As reach
// through to it.
func (e *Error) Unwrap() error { return e.err }

// The Bolt::Error tree. Each value maps one bbolt sentinel onto the Ruby
// exception class a host should raise.
var (
	ErrDatabaseNotOpen    = &Error{Class: "Bolt::DatabaseNotOpen", err: bolt.ErrDatabaseNotOpen}
	ErrInvalid            = &Error{Class: "Bolt::Invalid", err: bolt.ErrInvalid}
	ErrInvalidMapping     = &Error{Class: "Bolt::InvalidMapping", err: bolt.ErrInvalidMapping}
	ErrVersionMismatch    = &Error{Class: "Bolt::VersionMismatch", err: bolt.ErrVersionMismatch}
	ErrChecksum           = &Error{Class: "Bolt::Checksum", err: bolt.ErrChecksum}
	ErrTimeout            = &Error{Class: "Bolt::Timeout", err: bolt.ErrTimeout}
	ErrTxNotWritable      = &Error{Class: "Bolt::TxNotWritable", err: bolt.ErrTxNotWritable}
	ErrTxClosed           = &Error{Class: "Bolt::TxClosed", err: bolt.ErrTxClosed}
	ErrDatabaseReadOnly   = &Error{Class: "Bolt::DatabaseReadOnly", err: bolt.ErrDatabaseReadOnly}
	ErrFreePagesNotLoaded = &Error{Class: "Bolt::FreePagesNotLoaded", err: bolt.ErrFreePagesNotLoaded}
	ErrBucketNotFound     = &Error{Class: "Bolt::BucketNotFound", err: bolt.ErrBucketNotFound}
	ErrBucketExists       = &Error{Class: "Bolt::BucketExists", err: bolt.ErrBucketExists}
	ErrBucketNameRequired = &Error{Class: "Bolt::BucketNameRequired", err: bolt.ErrBucketNameRequired}
	ErrKeyRequired        = &Error{Class: "Bolt::KeyRequired", err: bolt.ErrKeyRequired}
	ErrKeyTooLarge        = &Error{Class: "Bolt::KeyTooLarge", err: bolt.ErrKeyTooLarge}
	ErrValueTooLarge      = &Error{Class: "Bolt::ValueTooLarge", err: bolt.ErrValueTooLarge}
	ErrIncompatibleValue  = &Error{Class: "Bolt::IncompatibleValue", err: bolt.ErrIncompatibleValue}
)

// errorTable maps each bbolt sentinel to its Bolt::Error. mapError walks it.
var errorTable = []*Error{
	ErrDatabaseNotOpen,
	ErrInvalid,
	ErrInvalidMapping,
	ErrVersionMismatch,
	ErrChecksum,
	ErrTimeout,
	ErrTxNotWritable,
	ErrTxClosed,
	ErrDatabaseReadOnly,
	ErrFreePagesNotLoaded,
	ErrBucketNotFound,
	ErrBucketExists,
	ErrBucketNameRequired,
	ErrKeyRequired,
	ErrKeyTooLarge,
	ErrValueTooLarge,
	ErrIncompatibleValue,
}

// mapError translates a go.etcd.io/bbolt sentinel into the corresponding
// Bolt::Error. A nil error maps to nil; an error that is not a known bbolt
// sentinel (for example a caller's own error returned from an Update block to
// force a rollback) is returned unchanged so the caller sees exactly what they
// threw.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	for _, e := range errorTable {
		if errors.Is(err, e.err) {
			return e
		}
	}
	return err
}
