<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-bbolt/brand/main/social/go-ruby-bbolt-bbolt.png" alt="go-ruby-bbolt/bbolt" width="720"></p>

# bbolt — go-ruby-bbolt

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-bbolt.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A clean, idiomatic Ruby-style key/value store API over
[`go.etcd.io/bbolt`](https://pkg.go.dev/go.etcd.io/bbolt) — the pure-Go, embedded,
ACID B+tree store.**

bbolt is Go-native: there is no canonical Ruby `bbolt` gem to reimplement. This
module instead designs the surface a Ruby **`Bolt`** gem *would* have — buckets,
ACID transactions, and cursors — the way a Rubyist expects to use an embedded
store, and implements it directly on top of the real bbolt engine. It exposes a
Go capability as an idiomatic Ruby API: it sits alongside Ruby's own
[`PStore`](https://docs.ruby-lang.org/en/master/PStore.html) and `GDBM` idioms,
but, unlike them, offers genuine **MVCC ACID transactions** — one writer and many
concurrent readers over a single memory-mapped file, with commit/rollback.

The storage engine — the copy-on-write B+tree, the page cache, the freelist, the
mmap, and the write-ahead crash safety — is **bbolt's and is not reimplemented
here**. This module is a thin, allocation-conscious façade that renames bbolt's
Go surface to the shape a Ruby binding wants and maps bbolt's sentinel errors
onto a `Bolt::Error` class tree the host can raise. The engine is pure Go, so the
whole stack links statically with `CGO_ENABLED=0` on every 64-bit target the
go-\* ecosystem supports, and on Linux, macOS and Windows.

It is a sibling of
[go-ruby-sqlite3](https://github.com/go-ruby-sqlite3/sqlite3) (the pure-Go
SQLite backend) and [go-ruby-pstore](https://github.com/go-ruby-pstore/pstore)
(the PStore port), and is a **standalone, reusable** module for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby).

## Ruby surface

| Ruby (`Bolt`)                             | Go                              |
| ----------------------------------------- | ------------------------------- |
| `Bolt::DB.open(path, options)`            | `Open(path, opts)`              |
| `Bolt.open(path)`                         | `Open(path, nil)`               |
| `db.close` / `db.path` / `db.stats`       | `DB.Close` / `Path` / `Stats`   |
| `db.update { \|tx\| … }`                  | `DB.Update` (read-write)        |
| `db.view { \|tx\| … }`                    | `DB.View` (read-only)           |
| `db.begin(writable)`                      | `DB.Begin` → `Tx.Commit`/`Rollback` |
| `tx.bucket(name)` / `create_bucket` …     | `Tx.Bucket` / `CreateBucket` …  |
| `tx.buckets`                              | `Tx.Buckets`                    |
| `bucket.put/get/delete`                   | `Bucket.Put` / `Get` / `Delete` |
| `bucket.each { \|k, v\| … }`              | `Bucket.Each`                   |
| `bucket.cursor`                           | `Bucket.Cursor`                 |
| `bucket.bucket(name)` (nested)            | `Bucket.Bucket` / `CreateBucket`|
| `bucket.next_sequence`                    | `Bucket.NextSequence`           |
| `cursor.first/next/prev/last/seek/delete` | `Cursor.First` / `Next` / …     |

Keys and values are `[]byte` throughout — a Ruby host maps them to and from
`ASCII-8BIT` (binary) Strings. Using `[]byte` (rather than `string`) keeps binary
safety and preserves bbolt's distinction between a **missing** key (`Get` returns
a `nil` slice) and a present, **empty** value (a non-nil, zero-length slice).

## Install

```sh
go get github.com/go-ruby-bbolt/bbolt
```

## Usage

```go
package main

import (
	"fmt"

	bbolt "github.com/go-ruby-bbolt/bbolt"
)

func main() {
	db, _ := bbolt.Open("app.db", nil) // Bolt.open("app.db")
	defer db.Close()

	// Read-write transaction (Bolt::DB#update): commits on success,
	// rolls back on any returned error or panic.
	db.Update(func(tx *bbolt.Tx) error {
		users, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}
		id, _ := users.NextSequence()
		return users.Put([]byte(fmt.Sprintf("%d", id)), []byte("alice"))
	})

	// Read-only transaction (Bolt::DB#view) with a cursor.
	db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte("users")).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("%s = %s\n", k, v)
		}
		return nil
	})
}
```

## Error tree

Errors are `*bbolt.Error`, each naming the `Bolt::` exception class a host should
raise and wrapping the underlying bbolt sentinel, so both this package's sentinel
and bbolt's own report through `errors.Is`:

```go
err := db.Update(func(tx *bbolt.Tx) error {
	_, err := tx.CreateBucket([]byte("dup"))
	if err != nil {
		return err
	}
	_, err = tx.CreateBucket([]byte("dup")) // already exists
	return err
})

var e *bbolt.Error
if errors.As(err, &e) {
	fmt.Println(e.Class) // Bolt::BucketExists
}
errors.Is(err, bbolt.ErrBucketExists)       // true (this package)
errors.Is(err, boltpkg.ErrBucketExists)     // true (go.etcd.io/bbolt)
```

The tree covers `Bolt::DatabaseNotOpen`, `BucketExists`, `BucketNotFound`,
`BucketNameRequired`, `TxClosed`, `TxNotWritable`, `DatabaseReadOnly`,
`KeyRequired`, `KeyTooLarge`, `ValueTooLarge`, `IncompatibleValue`, `Timeout`,
`Invalid`, `VersionMismatch`, `Checksum`, and more — one per bbolt sentinel. A
caller's own error returned from an `Update` block to force a rollback is passed
back unchanged.

## Backend & architectures

The engine is `go.etcd.io/bbolt` — a real, embedded, ACID B+tree, **no cgo**.
Every arch below builds and tests with `CGO_ENABLED=0`:

| arch    | CGO=0 build | notes                          |
| ------- | ----------- | ------------------------------ |
| amd64   | ✅          | native CI lane                 |
| arm64   | ✅          | native CI lane                 |
| riscv64 | ✅          | qemu-user CI lane              |
| loong64 | ✅          | qemu-user CI lane              |
| ppc64le | ✅          | qemu-user CI lane              |
| s390x   | ✅          | qemu-user CI lane (big-endian) |

The `-race` host lane keeps the default toolchain (cgo on for the race detector);
the backend is pure-Go either way. Windows is a first-class host lane — bbolt's
mmap layer supports it.

### Cross-endian file-portability caveat

bbolt stores a **native-endian** magic number in its file header and validates it
on open, so a database file is **not portable across byte orders**: a file
written on a little-endian host (amd64/arm64/riscv64/loong64/ppc64le) will fail
to open on a big-endian host (s390x), and vice versa. This is a property of the
bbolt on-disk format, not of this binding, and it is not a problem here: every
test writes and reads a fresh temp-file database on the **same** host, so the
format round-trips cleanly and the suite is green on s390x (big-endian) as well
as the little-endian arches. If you need a portable dump, serialise the logical
key/value contents rather than copying the file.

## Tests & coverage

Deterministic, interpreter-free tests over temp-file databases (`t.TempDir`)
exercise put/get/delete/iterate in read-write and read-only transactions, nested
buckets, cursor seek/first/last/next/prev, sequences, rollback-discards /
commit-persists (verified by reopening the DB), concurrent read transactions, and
the full error surface (bucket exists/not-found, tx closed, read-only writes).
They alone hold coverage at **100%**, so the qemu cross-arch and Windows lanes
pass the gate with no network and no external oracle.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-bbolt/bbolt authors.

## WebAssembly

Unlike the rest of the go-ruby family, this library does **not** target WebAssembly:
its backing engine — go.etcd.io/bbolt (memory-mapped files + file locking) — relies on `mmap` and native filesystem syscalls that
the `js/wasm` and `wasip1/wasm` sandboxes do not provide. It ships for the six
64-bit native/qemu arches only.
