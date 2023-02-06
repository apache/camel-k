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

package indentedwriter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Flusher --.
type Flusher interface {
	Flush()
}

// Writer --.
type Writer struct {
	out io.Writer
}

// NewWriter --.
func NewWriter(out io.Writer) *Writer {
	return &Writer{out: out}
}

// Writef --.
func (iw *Writer) Writef(indentLevel int, format string, i ...interface{}) {
	fmt.Fprint(iw.out, strings.Repeat("  ", indentLevel))
	fmt.Fprintf(iw.out, format, i...)
}

// Writelnf --.
func (iw *Writer) Writelnf(indentLevel int, format string, i ...interface{}) {
	fmt.Fprint(iw.out, strings.Repeat("  ", indentLevel))
	fmt.Fprintf(iw.out, format, i...)
	fmt.Fprint(iw.out, "\n")
}

// Flush --.
func (iw *Writer) Flush() {
	if f, ok := iw.out.(Flusher); ok {
		f.Flush()
	}
}

// IndentedString --.
func IndentedString(f func(io.Writer) error) (string, error) {
	var out tabwriter.Writer
	buf := &bytes.Buffer{}
	out.Init(buf, 0, 8, 2, ' ', 0)

	err := f(&out)
	if err != nil {
		return "", err
	}

	err = out.Flush()
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
