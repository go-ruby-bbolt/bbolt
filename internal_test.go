package bbolt

import (
	"errors"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestMapErrorNil(t *testing.T) {
	if err := mapError(nil); err != nil {
		t.Fatalf("mapError(nil) = %v, want nil", err)
	}
}

func TestMapErrorKnownSentinels(t *testing.T) {
	// Every bbolt sentinel in the table must map to its Bolt::Error, and the
	// mapped value must report both its own identity and the bbolt sentinel via
	// errors.Is.
	for _, want := range errorTable {
		got := mapError(want.err)
		if got != want {
			t.Fatalf("mapError(%v) = %v, want %v", want.err, got, want)
		}
		be, ok := got.(*Error)
		if !ok {
			t.Fatalf("mapError(%v) is %T, want *Error", want.err, got)
		}
		if be.Class == "" {
			t.Fatalf("mapped error for %v has empty Class", want.err)
		}
		if !errors.Is(got, want.err) {
			t.Fatalf("errors.Is(mapped, %v) = false", want.err)
		}
		if !errors.Is(got, want) {
			t.Fatalf("errors.Is(mapped, sentinel) = false")
		}
	}
}

func TestMapErrorUnknownPassthrough(t *testing.T) {
	own := errors.New("caller error")
	if got := mapError(own); !errors.Is(got, own) {
		t.Fatalf("mapError(unknown) = %v, want passthrough of %v", got, own)
	}
}

func TestErrorMethods(t *testing.T) {
	e := ErrBucketNotFound
	if e.Error() != bolt.ErrBucketNotFound.Error() {
		t.Fatalf("Error() = %q, want %q", e.Error(), bolt.ErrBucketNotFound.Error())
	}
	if e.Unwrap() != bolt.ErrBucketNotFound {
		t.Fatalf("Unwrap() = %v, want bolt.ErrBucketNotFound", e.Unwrap())
	}
	if e.Class != "Bolt::BucketNotFound" {
		t.Fatalf("Class = %q", e.Class)
	}
}
