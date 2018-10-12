package blush

import (
	"bufio"
	"io"

	"github.com/arsham/blush/internal/reader"
)

type mode int

const (
	// Separator string between name of the reader and the contents.
	Separator = ": "

	// DefaultLineCache is minimum lines to cache.
	DefaultLineCache = 50

	// DefaultCharCache is minimum characters to cache for each line. This is in
	// effect only if Read() function is used.
	DefaultCharCache = 1000

	readMode mode = iota
	writeToMode
)

// Blush reads from reader and matches against all finders. If NoCut is true,
// any unmatched lines are printed as well. If WithFileName is true, blush will
// write the filename before it writes the output. Read and WriteTo will return
// ErrReadWriteMix if both Read and WriteTo are called on the same object. See
// package docs for more details.
type Blush struct {
	Finders      []Finder
	Reader       io.ReadCloser
	LineCache    uint
	CharCache    uint
	NoCut        bool // do not cut out non-matched lines.
	WithFileName bool
	closed       bool
	readLineCh   chan []byte
	readCh       chan byte
	mode         mode
}

// Read creates a goroutine on first invocation to read from the underlying
// reader. It is considerably slower than WriteTo as it reads the bytes one by
// one in order to produce the results, therefore you should use WriteTo
// directly or use io.Copy() on blush.
func (b *Blush) Read(p []byte) (n int, err error) {
	if b.closed {
		return 0, ErrClosed
	}
	if b.mode == writeToMode {
		return 0, ErrReadWriteMix
	}
	if b.mode != readMode {
		if err = b.setup(readMode); err != nil {
			return 0, err
		}
	}
	for n = 0; n < cap(p); n++ {
		c, ok := <-b.readCh
		if !ok {
			return n, io.EOF
		}
		p[n] = c
	}
	return n, err
}

// WriteTo writes matches to w. It returns an error if the writer is nil or
// there are not paths defined or there is no files found in the Reader.
func (b *Blush) WriteTo(w io.Writer) (int64, error) {
	if b.closed {
		return 0, ErrClosed
	}
	if b.mode == readMode {
		return 0, ErrReadWriteMix
	}
	if b.mode != writeToMode {
		if err := b.setup(writeToMode); err != nil {
			return 0, err
		}
	}
	var total int
	if w == nil {
		return 0, ErrNoWriter
	}
	for line := range b.readLineCh {
		if n, err := w.Write(line); err != nil {
			return int64(n), err
		}
		total += len(line)
	}
	return int64(total), nil
}

func (b *Blush) setup(m mode) error {
	if b.Reader == nil {
		return reader.ErrNoReader
	}
	if len(b.Finders) < 1 {
		return ErrNoFinder
	}

	b.mode = m
	if b.LineCache == 0 {
		b.LineCache = DefaultLineCache
	}
	if b.CharCache == 0 {
		b.CharCache = DefaultCharCache
	}
	b.readLineCh = make(chan []byte, b.LineCache)
	b.readCh = make(chan byte, b.CharCache)
	go b.readLines()
	if m == readMode {
		go b.transfer()
	}
	return nil
}

func (b Blush) decorate(input string) (string, bool) {
	str, ok := lookInto(b.Finders, input)
	if ok || b.NoCut {
		var prefix string
		if b.WithFileName {
			prefix = fileName(b.Reader)
		}
		return prefix + str, true
	}
	return "", false
}

func (b Blush) readLines() {
	var (
		ok bool
		sc = bufio.NewReader(b.Reader)
	)
	for {
		line, err := sc.ReadString('\n')
		if line, ok = b.decorate(line); ok {
			b.readLineCh <- []byte(line)
		}
		if err != nil {
			break
		}
	}
	close(b.readLineCh)
}

func (b Blush) transfer() {
	for line := range b.readLineCh {
		for _, c := range line {
			b.readCh <- c
		}
	}
	close(b.readCh)
}

// Close closes the reader and returns whatever error it returns.
func (b *Blush) Close() error {
	b.closed = true
	return b.Reader.Close()
}

// lookInto returns a new decorated line if any of the finders decorate it, or
// the given line as it is.
func lookInto(f []Finder, line string) (string, bool) {
	var found bool
	for _, a := range f {
		if s, ok := a.Find(line); ok {
			line = s
			found = true
		}
	}
	return line, found
}

// fileName returns an empty string if it could not query the fileName from r.
func fileName(r io.Reader) string {
	type namer interface {
		FileName() string
	}
	if o, ok := r.(namer); ok {
		return o.FileName() + Separator
	}
	return ""
}
