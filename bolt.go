package bbolt

import (
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
)

// DB is an open Bolt database — the Ruby Bolt::DB. It wraps a single
// memory-mapped bbolt file and hands out read-write and read-only transactions.
type DB struct {
	db   *bolt.DB
	path string
}

// Options configures Open, mirroring the option hash a Ruby Bolt::DB.open would
// accept. The zero value is valid and yields a read-write database created with
// mode 0600.
type Options struct {
	// Mode is the file mode used when the database file is created. Zero means
	// 0600.
	Mode os.FileMode

	// ReadOnly opens the database in read-only mode, taking a shared lock. Write
	// transactions then fail with ErrDatabaseReadOnly.
	ReadOnly bool

	// Timeout is how long Open waits to obtain the file lock. Zero waits
	// indefinitely; on a locked file a non-zero timeout fails with ErrTimeout.
	Timeout time.Duration

	// NoGrowSync sets bbolt's NoGrowSync flag before memory-mapping the file.
	NoGrowSync bool

	// NoFreelistSync avoids syncing the freelist to disk, trading recovery cost
	// for write throughput.
	NoFreelistSync bool
}

// Open opens (creating if necessary) the Bolt database at path. It is both
// Bolt::DB.open(path, options) and, with a nil opts, Bolt.open(path).
func Open(path string, opts *Options) (*DB, error) {
	mode := os.FileMode(0o600)
	var bopts *bolt.Options
	if opts != nil {
		if opts.Mode != 0 {
			mode = opts.Mode
		}
		bopts = &bolt.Options{
			ReadOnly:       opts.ReadOnly,
			Timeout:        opts.Timeout,
			NoGrowSync:     opts.NoGrowSync,
			NoFreelistSync: opts.NoFreelistSync,
		}
	}
	db, err := bolt.Open(path, mode, bopts)
	if err != nil {
		return nil, mapError(err)
	}
	return &DB{db: db, path: path}, nil
}

// Path returns the path to the database file (Bolt::DB#path).
func (d *DB) Path() string { return d.path }

// IsReadOnly reports whether the database was opened read-only.
func (d *DB) IsReadOnly() bool { return d.db.IsReadOnly() }

// Close releases all resources and unmaps the file (Bolt::DB#close).
func (d *DB) Close() error { return mapError(d.db.Close()) }

// Update runs fn inside a read-write transaction (Bolt::DB#update). If fn
// returns nil the transaction is committed; if it returns an error or panics,
// the transaction is rolled back and the error is returned unchanged.
func (d *DB) Update(fn func(*Tx) error) error {
	return mapError(d.db.Update(func(btx *bolt.Tx) error {
		return fn(&Tx{tx: btx})
	}))
}

// View runs fn inside a read-only transaction (Bolt::DB#view). Writes attempted
// inside fn fail with ErrTxNotWritable.
func (d *DB) View(fn func(*Tx) error) error {
	return mapError(d.db.View(func(btx *bolt.Tx) error {
		return fn(&Tx{tx: btx})
	}))
}

// Begin starts an explicit transaction (Bolt::DB#begin). Pass writable=true for
// a read-write transaction. The caller must eventually call Tx.Commit or
// Tx.Rollback. Only one writable transaction may be open at a time; any number
// of read-only transactions may run concurrently.
func (d *DB) Begin(writable bool) (*Tx, error) {
	btx, err := d.db.Begin(writable)
	if err != nil {
		return nil, mapError(err)
	}
	return &Tx{tx: btx}, nil
}

// Stats is a snapshot of database-level counters (Bolt::DB#stats).
type Stats struct {
	// TxN is the total number of started read transactions.
	TxN int
	// OpenTxN is the number of currently open read transactions.
	OpenTxN int
	// FreePageN is the number of free pages on the freelist.
	FreePageN int
	// PendingPageN is the number of pending pages on the freelist.
	PendingPageN int
	// FreeAlloc is the bytes allocated in free pages.
	FreeAlloc int
	// FreelistInuse is the bytes used by the freelist.
	FreelistInuse int
}

// Stats returns a snapshot of database-level statistics (Bolt::DB#stats).
func (d *DB) Stats() Stats {
	s := d.db.Stats()
	return Stats{
		TxN:           s.TxN,
		OpenTxN:       s.OpenTxN,
		FreePageN:     s.FreePageN,
		PendingPageN:  s.PendingPageN,
		FreeAlloc:     s.FreeAlloc,
		FreelistInuse: s.FreelistInuse,
	}
}
