package bbolt

import bolt "go.etcd.io/bbolt"

// Bucket is a Bolt bucket — the Ruby Bolt::Bucket — a named, ordered collection
// of key/value pairs that may also hold nested sub-buckets.
type Bucket struct {
	b *bolt.Bucket
}

// Writable reports whether the bucket belongs to a writable transaction.
func (b *Bucket) Writable() bool { return b.b.Writable() }

// Get returns the value for key, or a nil slice if the key is absent
// (Bolt::Bucket#get). A present but empty value is a non-nil, zero-length slice.
// The returned slice is only valid for the life of the transaction and must not
// be mutated.
func (b *Bucket) Get(key []byte) []byte { return b.b.Get(key) }

// Put stores value under key (Bolt::Bucket#put). An empty key fails with
// ErrKeyRequired; a read-only transaction fails with ErrTxNotWritable.
func (b *Bucket) Put(key, value []byte) error { return mapError(b.b.Put(key, value)) }

// Delete removes key from the bucket, if present (Bolt::Bucket#delete).
func (b *Bucket) Delete(key []byte) error { return mapError(b.b.Delete(key)) }

// Each iterates every key/value pair in byte order (Bolt::Bucket#each). If fn
// returns an error, iteration stops and that error is returned. Keys that map to
// nested buckets are yielded with a nil value.
func (b *Bucket) Each(fn func(key, value []byte) error) error {
	return mapError(b.b.ForEach(func(k, v []byte) error {
		return fn(k, v)
	}))
}

// Cursor returns a cursor over the bucket (Bolt::Bucket#cursor).
func (b *Bucket) Cursor() *Cursor { return &Cursor{c: b.b.Cursor()} }

// Bucket returns the nested sub-bucket with the given name, or nil if it does
// not exist (Bolt::Bucket#bucket).
func (b *Bucket) Bucket(name []byte) *Bucket {
	sub := b.b.Bucket(name)
	if sub == nil {
		return nil
	}
	return &Bucket{b: sub}
}

// CreateBucket creates a nested sub-bucket (Bolt::Bucket#create_bucket). It fails
// with ErrBucketExists if the sub-bucket already exists.
func (b *Bucket) CreateBucket(name []byte) (*Bucket, error) {
	sub, err := b.b.CreateBucket(name)
	if err != nil {
		return nil, mapError(err)
	}
	return &Bucket{b: sub}, nil
}

// CreateBucketIfNotExists creates a nested sub-bucket if it does not already
// exist (Bolt::Bucket#create_bucket_if_not_exists).
func (b *Bucket) CreateBucketIfNotExists(name []byte) (*Bucket, error) {
	sub, err := b.b.CreateBucketIfNotExists(name)
	if err != nil {
		return nil, mapError(err)
	}
	return &Bucket{b: sub}, nil
}

// DeleteBucket removes a nested sub-bucket (Bolt::Bucket#delete_bucket).
func (b *Bucket) DeleteBucket(name []byte) error { return mapError(b.b.DeleteBucket(name)) }

// Buckets returns the names of the immediate nested sub-buckets, in byte order
// (Bolt::Bucket#buckets). Each returned slice is a copy owned by the caller.
func (b *Bucket) Buckets() [][]byte {
	var names [][]byte
	_ = b.b.ForEachBucket(func(k []byte) error {
		names = append(names, append([]byte(nil), k...))
		return nil
	})
	return names
}

// Sequence returns the current monotonic sequence counter (Bolt::Bucket#sequence).
func (b *Bucket) Sequence() uint64 { return b.b.Sequence() }

// SetSequence sets the sequence counter (Bolt::Bucket#sequence=).
func (b *Bucket) SetSequence(v uint64) error { return mapError(b.b.SetSequence(v)) }

// NextSequence returns a new, auto-incrementing sequence value unique within the
// bucket (Bolt::Bucket#next_sequence).
func (b *Bucket) NextSequence() (uint64, error) {
	n, err := b.b.NextSequence()
	return n, mapError(err)
}
