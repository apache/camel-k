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
	"text/tabwriter"
)

// Flusher --
type Flusher interface {
	Flush()
}

// Writer --.
type Writer struct {
	out io.Writer
}

// NewWriter --
func NewWriter(out io.Writer) *Writer {
	return &Writer{out: out}
}

// Write --
func (iw *Writer) Write(indentLevel int, format string, i ...interface{}) {
	indent := "  "
	prefix := ""
	for i := 0; i < indentLevel; i++ {
		prefix += indent
	}
	fmt.Fprintf(iw.out, prefix+format, i...)
}

// Flush --
func (iw *Writer) Flush() {
	if f, ok := iw.out.(Flusher); ok {
		f.Flush()
	}
}

// IndentedString --
func IndentedString(f func(io.Writer)) string {
	var out tabwriter.Writer
	buf := &bytes.Buffer{}
	out.Init(buf, 0, 8, 2, ' ', 0)

	f(&out)

	out.Flush()

	return buf.String()
}
