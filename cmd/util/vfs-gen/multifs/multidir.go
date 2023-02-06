/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Based on the union fs function available at
https://github.com/shurcooL/httpfs/blob/master/union/union.go
(Licenced under MIT)
*/

package multifs

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shurcooL/httpfs/vfsutil"
)

func New(rootDir string, dirNames []string, exclude []string) (http.FileSystem, error) {
	m := &multiFS{
		rootDir: rootDir,
		exclude: exclude,
		mfs:     make(map[string]http.FileSystem),
		root: &dirInfo{
			name: "/",
		},
	}
	for _, dirName := range dirNames {
		err := m.bind(dirName)
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

type multiFS struct {
	rootDir string
	exclude []string
	mfs     map[string]http.FileSystem
	root    *dirInfo
}

func (m *multiFS) bind(dirName string) error {
	absDir := filepath.Join(m.rootDir, dirName)

	hfs := http.Dir(absDir)
	m.mfs["/"+dirName] = hfs

	//
	// The 1-level down paths are needed since the
	// remainder are covered by the http filesystems
	//
	fileInfos, err := vfsutil.ReadDir(hfs, "/")
	if err != nil {
		return err
	}

	for _, nfo := range fileInfos {
		path := "/" + nfo.Name()

		if m.excluded(path) {
			continue // skip
		}

		if nfo.IsDir() {
			m.root.entries = append(m.root.entries, &dirInfo{
				name: path,
			})
		} else {
			m.root.entries = append(m.root.entries, nfo)
		}
	}

	return nil
}

func (m *multiFS) excluded(path string) bool {
	for _, ex := range m.exclude {
		if strings.HasPrefix(path, ex) {
			return true
		}
	}

	return false
}

func (m *multiFS) Open(path string) (http.File, error) {
	if path == "/" {
		return &dir{
			dirInfo: m.root,
		}, nil
	}

	for _, fs := range m.mfs {
		f, err := fs.Open(path)
		if err != nil {
			continue
		}

		return f, nil
	}

	return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
}

// dirInfo is a static definition of a directory.
type dirInfo struct {
	name    string
	entries []os.FileInfo
}

func (d *dirInfo) Read([]byte) (int, error) {
	return 0, fmt.Errorf("cannot Read from directory %s", d.name)
}
func (d *dirInfo) Close() error               { return nil }
func (d *dirInfo) Stat() (os.FileInfo, error) { return d, nil }

func (d *dirInfo) Name() string       { return d.name }
func (d *dirInfo) Size() int64        { return 0 }
func (d *dirInfo) Mode() os.FileMode  { return 0o755 | os.ModeDir }
func (d *dirInfo) ModTime() time.Time { return time.Time{} } // Actual mod time is not computed because it's expensive and rarely needed.
func (d *dirInfo) IsDir() bool        { return true }
func (d *dirInfo) Sys() interface{}   { return nil }

// dir is an opened dir instance.
type dir struct {
	*dirInfo
	pos int // Position within entries for Seek and Readdir.
}

func (d *dir) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && whence == io.SeekStart {
		d.pos = 0
		return 0, nil
	}
	return 0, fmt.Errorf("unsupported Seek in directory %s", d.dirInfo.name)
}

func (d *dir) Readdir(count int) ([]os.FileInfo, error) {
	if d.pos >= len(d.dirInfo.entries) && count > 0 {
		return nil, io.EOF
	}
	if count <= 0 || count > len(d.dirInfo.entries)-d.pos {
		count = len(d.dirInfo.entries) - d.pos
	}
	e := d.dirInfo.entries[d.pos : d.pos+count]
	d.pos += count

	return e, nil
}
