package reader

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/arsham/blush/internal/tools"
	"github.com/pkg/errors"
)

// ErrNoReader is returned if there is no reader defined.
var ErrNoReader = errors.New("no input")

// MultiReader holds one or more io.ReadCloser and reads their contents when
// Read() method is called in order. The reader is loaded lazily if it is a
// file to prevent the system going out of file descriptors.
type MultiReader struct {
	readers     []*container
	currentName string
}

// NewMultiReader creates an instance of the MultiReader and passes it to all
// input functions.
func NewMultiReader(input ...Conf) (*MultiReader, error) {
	m := &MultiReader{
		readers: make([]*container, 0),
	}
	for _, c := range input {
		if c == nil {
			return nil, ErrNoReader
		}
		err := c(m)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Conf is used to configure the MultiReader.
type Conf func(*MultiReader) error

// WithReader adds the {name,r} reader to the MultiReader. If name is empty, the
// key will not be written in the output. You can provide as many empty names as
// you need.
func WithReader(name string, r io.ReadCloser) Conf {
	return func(m *MultiReader) error {
		if r == nil {
			return errors.Wrap(ErrNoReader, "WithReader")
		}
		c := &container{
			get: func() (io.ReadCloser, error) {
				m.currentName = name
				return r, nil
			},
		}
		m.readers = append(m.readers, c)
		return nil
	}
}

// WithPaths searches through the path and adds any files it finds to the
// MultiReader. Each path will become its reader's name in the process. It
// returns an error if any of given files are not found. It ignores any files
// that cannot be read or opened.
func WithPaths(paths []string, recursive bool) Conf {
	return func(m *MultiReader) error {
		if paths == nil {
			return errors.Wrap(ErrNoReader, "WithPaths: nil paths")
		}
		if len(paths) == 0 {
			return errors.Wrap(ErrNoReader, "WithPaths: empty paths")
		}
		files, err := tools.Files(recursive, paths...)
		if err != nil {
			return errors.Wrap(err, "WithPaths")
		}
		for _, name := range files {
			name := name
			c := &container{
				get: func() (io.ReadCloser, error) {
					m.currentName = name
					f, err := os.Open(name)
					return f, err
				},
			}
			m.readers = append(m.readers, c)
		}
		return nil
	}
}

// Read is almost the exact implementation of io.MultiReader but keeps track of
// reader names. It closes each reader once they report they are exhausted, and
// it will happen on the next read.
func (m *MultiReader) Read(b []byte) (n int, err error) {
	for len(m.readers) > 0 {
		if len(m.readers) == 1 {
			if r, ok := m.readers[0].r.(*MultiReader); ok {
				m.readers = r.readers
				continue
			}
		}
		n, err = m.readers[0].Read(b)
		if err == io.EOF {
			m.readers[0].r.Close()
			c := &container{r: ioutil.NopCloser(nil)}
			m.readers[0] = c
			m.readers = m.readers[1:]
		}
		if n > 0 || err != io.EOF {
			if err == io.EOF && len(m.readers) > 0 {
				err = nil
			}
			return
		}
	}
	m.currentName = ""
	return 0, io.EOF
}

// Close does nothing.
func (m *MultiReader) Close() error { return nil }

// FileName returns the current reader's name.
func (m *MultiReader) FileName() string {
	return m.currentName
}

// container takes care of opening the reader on demand. This is particularly
// useful when searching in thousands of files, because we want to open them on
// demand, otherwise the system gets out of file descriptors.
type container struct {
	r    io.ReadCloser
	open bool
	get  func() (io.ReadCloser, error)
}

func (c *container) Read(b []byte) (int, error) {
	if !c.open {
		var err error
		c.r, err = c.get()
		if err != nil {
			return 0, err
		}
		c.open = true
	}
	return c.r.Read(b)
}
