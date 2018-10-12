package blush

import "errors"

// ErrNoWriter is returned if a nil object is passed to the WriteTo method.
var ErrNoWriter = errors.New("no writer defined")

// ErrNoFinder is returned if there is no finder passed to Blush.
var ErrNoFinder = errors.New("no finders defined")

// ErrClosed is returned if the reader is closed and you try to read from it.
var ErrClosed = errors.New("reader already closed")

// ErrReadWriteMix is returned when the Read and WriteTo are called on the same
// object.
var ErrReadWriteMix = errors.New("you cannot mix Read and WriteTo calls")
