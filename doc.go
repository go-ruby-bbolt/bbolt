// Package bbolt exposes [go.etcd.io/bbolt] — the pure-Go, embedded, ACID
// key/value store — as a clean, idiomatic Ruby-style API for the Ruby / rbgo
// ecosystem.
//
// # What it is — and isn't
//
// bbolt is Go-native: there is no canonical Ruby "bbolt" gem to reimplement.
// This module instead designs the surface a Ruby "Bolt" gem *would* have —
// buckets, ACID transactions, and cursors — the way a Rubyist expects to use an
// embedded store, and implements it directly on top of the real bbolt B+tree
// engine. It sits alongside Ruby's own persistence idioms (the standard
// library's PStore and GDBM) but, unlike them, offers genuine multi-version
// concurrency-control ACID transactions: one writer and many concurrent readers
// over a single memory-mapped file, with commit/rollback semantics.
//
// The storage engine — the copy-on-write B+tree, the page cache, the freelist,
// the mmap, and the write-ahead crash safety — is bbolt's and is *not*
// reimplemented here. This package is a thin, faithful, allocation-conscious
// façade that renames bbolt's Go surface to the shape a Ruby binding wants and
// maps bbolt's sentinel errors onto a Bolt::Error class tree the host can raise.
//
// The engine is pure Go, so the whole stack links statically with
// CGO_ENABLED=0 on every 64-bit target the go-* ecosystem supports
// (amd64, arm64, riscv64, loong64, ppc64le, s390x), and on Linux, macOS and
// Windows (bbolt's mmap layer supports all three).
//
// # Ruby surface
//
//	Bolt::DB.open(path, options)   -> Open(path, opts)
//	Bolt.open(path)                -> Open(path, nil)
//	db.close / db.path / db.stats  -> DB.Close / DB.Path / DB.Stats
//	db.update { |tx| ... }         -> DB.Update  (read-write, auto commit/rollback)
//	db.view   { |tx| ... }         -> DB.View    (read-only)
//	db.begin(writable)             -> DB.Begin -> Tx.Commit / Tx.Rollback
//	tx.bucket / create_bucket ...  -> Tx.Bucket / CreateBucket / ...
//	bucket.put / get / delete      -> Bucket.Put / Get / Delete
//	bucket.each { |k, v| ... }     -> Bucket.Each
//	bucket.cursor                  -> Cursor.First/Next/Prev/Last/Seek/Delete
//	bucket.next_sequence           -> Bucket.NextSequence
//
// Keys and values are []byte throughout. A Ruby host maps them to and from
// ASCII-8BIT (binary) Strings; using []byte (rather than string) keeps binary
// safety and preserves bbolt's distinction between a missing key (Get returns a
// nil slice) and a present, empty value (a non-nil, zero-length slice).
//
// # Cross-endian file-portability caveat
//
// bbolt stores a native-endian magic number in its file header and validates it
// on open. A database file is therefore *not* portable across byte orders: a
// file written on a little-endian host (amd64, arm64, riscv64, loong64,
// ppc64le) will fail to open on a big-endian host (s390x), and vice versa, with
// a "version mismatch" / "invalid database" error. This is a property of the
// bbolt on-disk format, not of this binding. It is not a problem in practice
// here: every test writes and reads a fresh temp-file database on the *same*
// host, so the format round-trips cleanly, and the suite is green on s390x
// (big-endian) as well as the little-endian arches. If you need a portable
// dump, serialise the logical key/value contents rather than copying the file.
package bbolt
