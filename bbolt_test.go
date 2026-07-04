package bbolt_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	bbolt "github.com/go-ruby-bbolt/bbolt"
	bolt "go.etcd.io/bbolt"
)

// openTemp opens a fresh database in a per-test temp directory.
func openTemp(t *testing.T) *bbolt.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := bbolt.Open(path, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenNilOptionsAndPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	db, err := bbolt.Open(path, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if db.Path() != path {
		t.Fatalf("path = %q, want %q", db.Path(), path)
	}
	if db.IsReadOnly() {
		t.Fatal("fresh db should not be read-only")
	}
}

func TestOpenWithModeOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mode.db")
	db, err := bbolt.Open(path, &bbolt.Options{Mode: 0o644})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtimeSupportsMode() && info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %v, want 0644", info.Mode().Perm())
	}
}

// runtimeSupportsMode reports whether the OS honours unix file-permission bits
// (Windows does not), so the mode assertion is skipped there.
func runtimeSupportsMode() bool { return os.PathSeparator == '/' }

func TestOpenReadOnlyRejectsWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ro.db")
	// Create the file first; a read-only open of a missing file would fail.
	db, err := bbolt.Open(path, nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	ro, err := bbolt.Open(path, &bbolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("reopen ro: %v", err)
	}
	defer ro.Close()
	if !ro.IsReadOnly() {
		t.Fatal("expected read-only db")
	}
	if _, err := ro.Begin(true); !errors.Is(err, bbolt.ErrDatabaseReadOnly) {
		t.Fatalf("Begin(true) on ro db = %v, want ErrDatabaseReadOnly", err)
	}
}

func TestOpenError(t *testing.T) {
	// A path under a non-existent directory cannot be created.
	bad := filepath.Join(t.TempDir(), "nope", "deeper", "x.db")
	if _, err := bbolt.Open(bad, nil); err == nil {
		t.Fatal("expected error opening under a missing directory")
	}
}

func TestUpdateCommitAndReopenPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")
	db, err := bbolt.Open(path, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("users"))
		if err != nil {
			return err
		}
		return b.Put([]byte("k"), []byte("v"))
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen: the committed data must still be there.
	db2, err := bbolt.Open(path, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	err = db2.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			t.Fatal("bucket vanished after reopen")
		}
		if got := b.Get([]byte("k")); !bytes.Equal(got, []byte("v")) {
			t.Fatalf("get = %q, want v", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestUpdateRollbackOnError(t *testing.T) {
	db := openTemp(t)
	sentinel := errors.New("abort")
	err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucket([]byte("b")); err != nil {
			return err
		}
		return sentinel // force rollback; must be returned unchanged
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("update err = %v, want sentinel", err)
	}
	// The bucket creation must have been rolled back.
	err = db.View(func(tx *bbolt.Tx) error {
		if tx.Bucket([]byte("b")) != nil {
			t.Fatal("bucket should not exist after rollback")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestViewWriteFails(t *testing.T) {
	db := openTemp(t)
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucket([]byte("b"))
		return err
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("b"))
		if err := b.Put([]byte("k"), []byte("v")); !errors.Is(err, bbolt.ErrTxNotWritable) {
			t.Fatalf("Put in view = %v, want ErrTxNotWritable", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestExplicitTransactionCommitRollback(t *testing.T) {
	db := openTemp(t)

	// Explicit writable tx, committed.
	tx, err := db.Begin(true)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if !tx.Writable() {
		t.Fatal("tx should be writable")
	}
	if tx.ID() < 0 {
		t.Fatalf("unexpected tx id %d", tx.ID())
	}
	b, err := tx.CreateBucket([]byte("b"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := b.Put([]byte("k"), []byte("v")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Explicit writable tx, rolled back — its change is discarded.
	tx2, err := db.Begin(true)
	if err != nil {
		t.Fatalf("begin2: %v", err)
	}
	if err := tx2.Bucket([]byte("b")).Put([]byte("k2"), []byte("v2")); err != nil {
		t.Fatalf("put2: %v", err)
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// Read-only explicit tx confirms only the committed key survives.
	rtx, err := db.Begin(false)
	if err != nil {
		t.Fatalf("begin ro: %v", err)
	}
	rb := rtx.Bucket([]byte("b"))
	if got := rb.Get([]byte("k")); !bytes.Equal(got, []byte("v")) {
		t.Fatalf("get k = %q, want v", got)
	}
	if rb.Get([]byte("k2")) != nil {
		t.Fatal("k2 should have been rolled back")
	}
	if err := rtx.Rollback(); err != nil {
		t.Fatalf("rollback ro: %v", err)
	}
}

func TestConcurrentReadTransactions(t *testing.T) {
	db := openTemp(t)
	if err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("b"))
		if err != nil {
			return err
		}
		return b.Put([]byte("k"), []byte("v"))
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.View(func(tx *bbolt.Tx) error {
				if got := tx.Bucket([]byte("b")).Get([]byte("k")); !bytes.Equal(got, []byte("v")) {
					t.Errorf("concurrent get = %q, want v", got)
				}
				return nil
			})
			if err != nil {
				t.Errorf("view: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestBucketKVAndMissingKey(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("b"))
		if err != nil {
			return err
		}
		if !b.Writable() {
			t.Fatal("bucket should be writable")
		}
		if err := b.Put([]byte("a"), []byte("1")); err != nil {
			return err
		}
		if b.Get([]byte("missing")) != nil {
			t.Fatal("missing key should return nil")
		}
		if err := b.Delete([]byte("a")); err != nil {
			return err
		}
		if b.Get([]byte("a")) != nil {
			t.Fatal("deleted key should return nil")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestBucketExistsAndNotFoundErrors(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucket([]byte("b")); err != nil {
			return err
		}
		if _, err := tx.CreateBucket([]byte("b")); !errors.Is(err, bbolt.ErrBucketExists) {
			t.Fatalf("re-create = %v, want ErrBucketExists", err)
		}
		if err := tx.DeleteBucket([]byte("absent")); !errors.Is(err, bbolt.ErrBucketNotFound) {
			t.Fatalf("delete absent = %v, want ErrBucketNotFound", err)
		}
		// Bucket lookup of a missing bucket returns nil.
		if tx.Bucket([]byte("absent")) != nil {
			t.Fatal("missing bucket should be nil")
		}
		// CreateBucketIfNotExists is idempotent; empty name is rejected.
		if _, err := tx.CreateBucketIfNotExists([]byte("b")); err != nil {
			t.Fatalf("cbine existing: %v", err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte("")); !errors.Is(err, bbolt.ErrBucketNameRequired) {
			t.Fatalf("cbine empty = %v, want ErrBucketNameRequired", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestBucketsListing(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		for _, n := range []string{"a", "b", "c"} {
			if _, err := tx.CreateBucket([]byte(n)); err != nil {
				return err
			}
		}
		got := tx.Buckets()
		want := []string{"a", "b", "c"}
		if len(got) != len(want) {
			t.Fatalf("buckets = %d, want %d", len(got), len(want))
		}
		for i, n := range want {
			if string(got[i]) != n {
				t.Fatalf("bucket[%d] = %q, want %q", i, got[i], n)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestNestedBuckets(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		root, err := tx.CreateBucket([]byte("root"))
		if err != nil {
			return err
		}
		child, err := root.CreateBucket([]byte("child"))
		if err != nil {
			return err
		}
		if err := child.Put([]byte("k"), []byte("v")); err != nil {
			return err
		}
		// Re-create nested -> exists.
		if _, err := root.CreateBucket([]byte("child")); !errors.Is(err, bbolt.ErrBucketExists) {
			t.Fatalf("nested re-create = %v, want ErrBucketExists", err)
		}
		// Idempotent nested creation + empty-name rejection.
		if _, err := root.CreateBucketIfNotExists([]byte("child")); err != nil {
			t.Fatalf("nested cbine: %v", err)
		}
		if _, err := root.CreateBucketIfNotExists([]byte("")); !errors.Is(err, bbolt.ErrBucketNameRequired) {
			t.Fatalf("nested cbine empty = %v, want ErrBucketNameRequired", err)
		}
		// Sub-bucket lookup: present and absent.
		if root.Bucket([]byte("child")) == nil {
			t.Fatal("child bucket should be found")
		}
		if root.Bucket([]byte("absent")) != nil {
			t.Fatal("absent sub-bucket should be nil")
		}
		// List sub-buckets.
		subs := root.Buckets()
		if len(subs) != 1 || string(subs[0]) != "child" {
			t.Fatalf("sub-buckets = %v, want [child]", subs)
		}
		// Delete sub-bucket.
		if err := root.DeleteBucket([]byte("child")); err != nil {
			t.Fatalf("delete child: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestEach(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("b"))
		if err != nil {
			return err
		}
		for _, k := range []string{"a", "b", "c"} {
			if err := b.Put([]byte(k), []byte(k+"!")); err != nil {
				return err
			}
		}
		// Full iteration.
		var seen []string
		if err := b.Each(func(k, v []byte) error {
			seen = append(seen, string(k)+"="+string(v))
			return nil
		}); err != nil {
			t.Fatalf("each: %v", err)
		}
		want := "a=a! b=b! c=c!"
		if got := joinSpace(seen); got != want {
			t.Fatalf("each saw %q, want %q", got, want)
		}
		// Early-exit error is propagated unchanged.
		stop := errors.New("stop")
		if err := b.Each(func(k, v []byte) error { return stop }); !errors.Is(err, stop) {
			t.Fatalf("each stop = %v, want stop", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

func joinSpace(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

func TestCursor(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("b"))
		if err != nil {
			return err
		}
		for _, k := range []string{"a", "b", "c", "d"} {
			if err := b.Put([]byte(k), []byte(k)); err != nil {
				return err
			}
		}
		c := b.Cursor()

		if k, _ := c.First(); string(k) != "a" {
			t.Fatalf("first = %q, want a", k)
		}
		if k, _ := c.Next(); string(k) != "b" {
			t.Fatalf("next = %q, want b", k)
		}
		if k, _ := c.Last(); string(k) != "d" {
			t.Fatalf("last = %q, want d", k)
		}
		if k, _ := c.Prev(); string(k) != "c" {
			t.Fatalf("prev = %q, want c", k)
		}
		if k, _ := c.Seek([]byte("c")); string(k) != "c" {
			t.Fatalf("seek c = %q, want c", k)
		}
		// Seek past the end returns a nil key.
		if k, _ := c.Seek([]byte("z")); k != nil {
			t.Fatalf("seek z = %q, want nil", k)
		}
		// Delete the entry at the cursor.
		c.Seek([]byte("b"))
		if err := c.Delete(); err != nil {
			t.Fatalf("cursor delete: %v", err)
		}
		if b.Get([]byte("b")) != nil {
			t.Fatal("b should be deleted")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestSequences(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("b"))
		if err != nil {
			return err
		}
		if b.Sequence() != 0 {
			t.Fatalf("initial sequence = %d, want 0", b.Sequence())
		}
		n1, err := b.NextSequence()
		if err != nil {
			return err
		}
		n2, err := b.NextSequence()
		if err != nil {
			return err
		}
		if n1 != 1 || n2 != 2 {
			t.Fatalf("sequences = %d,%d, want 1,2", n1, n2)
		}
		if err := b.SetSequence(100); err != nil {
			return err
		}
		if b.Sequence() != 100 {
			t.Fatalf("sequence = %d, want 100", b.Sequence())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

func TestStats(t *testing.T) {
	db := openTemp(t)
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucket([]byte("b"))
		return err
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	// A read transaction has been counted.
	if s := db.Stats(); s.TxN < 0 {
		t.Fatalf("unexpected stats: %+v", s)
	}
}

// TestBoltSentinelBridge confirms that errors.Is reaches through to the
// underlying go.etcd.io/bbolt sentinel as well as this package's own.
func TestBoltSentinelBridge(t *testing.T) {
	db := openTemp(t)
	err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucket([]byte("b")); err != nil {
			return err
		}
		_, err := tx.CreateBucket([]byte("b"))
		if !errors.Is(err, bolt.ErrBucketExists) {
			t.Fatalf("errors.Is(err, bolt.ErrBucketExists) = false for %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}
