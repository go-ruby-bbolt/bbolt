package bbolt

import bolt "go.etcd.io/bbolt"

// Cursor iterates the keys of a bucket in byte order — the Ruby Bolt::Cursor.
// A nil key returned by any of the movement methods means the cursor has moved
// past the end (or before the start) of the bucket. When a key names a nested
// sub-bucket, its value is nil.
type Cursor struct {
	c *bolt.Cursor
}

// First moves to the first key/value pair (Bolt::Cursor#first).
func (c *Cursor) First() (key, value []byte) { return c.c.First() }

// Last moves to the last key/value pair (Bolt::Cursor#last).
func (c *Cursor) Last() (key, value []byte) { return c.c.Last() }

// Next moves to the next key/value pair (Bolt::Cursor#next).
func (c *Cursor) Next() (key, value []byte) { return c.c.Next() }

// Prev moves to the previous key/value pair (Bolt::Cursor#prev).
func (c *Cursor) Prev() (key, value []byte) { return c.c.Prev() }

// Seek moves to the first key at or after seek (Bolt::Cursor#seek). If no such
// key exists it returns a nil key.
func (c *Cursor) Seek(seek []byte) (key, value []byte) { return c.c.Seek(seek) }

// Delete removes the key/value pair the cursor currently points at
// (Bolt::Cursor#delete).
func (c *Cursor) Delete() error { return mapError(c.c.Delete()) }
