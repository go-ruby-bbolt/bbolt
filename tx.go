package bbolt

import bolt "go.etcd.io/bbolt"

// Tx is a Bolt transaction — the Ruby Bolt::Tx. A writable transaction may
// create, delete and mutate buckets; a read-only one may only read. A Tx is not
// safe for concurrent use by multiple goroutines.
type Tx struct {
	tx *bolt.Tx
}

// Writable reports whether the transaction can perform writes.
func (tx *Tx) Writable() bool { return tx.tx.Writable() }

// ID returns the transaction id (Bolt::Tx#id).
func (tx *Tx) ID() int { return tx.tx.ID() }

// Commit writes all changes and closes the transaction (Bolt::Tx#commit).
// Committing a read-only transaction, or a transaction already finished, fails.
func (tx *Tx) Commit() error { return mapError(tx.tx.Commit()) }

// Rollback discards all changes and closes the transaction (Bolt::Tx#rollback).
func (tx *Tx) Rollback() error { return mapError(tx.tx.Rollback()) }

// Bucket returns the top-level bucket with the given name, or nil if it does not
// exist (Bolt::Tx#bucket).
func (tx *Tx) Bucket(name []byte) *Bucket {
	b := tx.tx.Bucket(name)
	if b == nil {
		return nil
	}
	return &Bucket{b: b}
}

// CreateBucket creates a new top-level bucket (Bolt::Tx#create_bucket). It fails
// with ErrBucketExists if the bucket already exists.
func (tx *Tx) CreateBucket(name []byte) (*Bucket, error) {
	b, err := tx.tx.CreateBucket(name)
	if err != nil {
		return nil, mapError(err)
	}
	return &Bucket{b: b}, nil
}

// CreateBucketIfNotExists creates a top-level bucket if it does not already
// exist (Bolt::Tx#create_bucket_if_not_exists).
func (tx *Tx) CreateBucketIfNotExists(name []byte) (*Bucket, error) {
	b, err := tx.tx.CreateBucketIfNotExists(name)
	if err != nil {
		return nil, mapError(err)
	}
	return &Bucket{b: b}, nil
}

// DeleteBucket removes a top-level bucket (Bolt::Tx#delete_bucket). It fails with
// ErrBucketNotFound if the bucket does not exist.
func (tx *Tx) DeleteBucket(name []byte) error {
	return mapError(tx.tx.DeleteBucket(name))
}

// Buckets returns the names of all top-level buckets, in byte order
// (Bolt::Tx#buckets). Each returned slice is a copy owned by the caller.
func (tx *Tx) Buckets() [][]byte {
	var names [][]byte
	// The closure only ever returns nil, so ForEach cannot fail here.
	_ = tx.tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
		names = append(names, append([]byte(nil), name...))
		return nil
	})
	return names
}
