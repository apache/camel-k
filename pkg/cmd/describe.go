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

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type flusher interface {
	flush()
}

type indentedWriter struct {
	out io.Writer
}

func newIndentedWriter(out io.Writer) *indentedWriter {
	return &indentedWriter{out: out}
}

func (iw *indentedWriter) write(indentLevel int, format string, i ...interface{}) {
	indent := "  "
	prefix := ""
	for i := 0; i < indentLevel; i++ {
		prefix += indent
	}
	fmt.Fprintf(iw.out, prefix+format, i...)
}

func (iw *indentedWriter) Flush() {
	if f, ok := iw.out.(flusher); ok {
		f.flush()
	}
}

func describeObjectMeta(w *indentedWriter, om metav1.ObjectMeta) {
	w.write(0, "Name:\t%s\n", om.Name)
	w.write(0, "Namespace:\t%s\n", om.Namespace)

	if len(om.GetLabels()) > 0 {
		w.write(0, "Labels:")
		for k, v := range om.Labels {
			w.write(0, "\t%s=%s\n", k, strings.TrimSpace(v))
		}
	}

	if len(om.GetAnnotations()) > 0 {
		w.write(0, "Annotations:")
		for k, v := range om.Annotations {
			w.write(0, "\t%s=%s\n", k, strings.TrimSpace(v))
		}
	}

	w.write(0, "Creation Timestamp:\t%s\n", om.CreationTimestamp.Format(time.RFC1123Z))
}

func describeTraits(w *indentedWriter, traits map[string]v1alpha1.TraitSpec) {
	if len(traits) > 0 {
		w.write(0, "Traits:\n")

		for trait := range traits {
			w.write(1, "%s:\n", strings.Title(trait))
			w.write(2, "Configuration:\n")
			for k, v := range traits[trait].Configuration {
				w.write(3, "%s:\t%s\n", strings.Title(k), v)
			}
		}
	}
}

func indentedString(f func(io.Writer)) string {
	out := new(tabwriter.Writer)
	buf := &bytes.Buffer{}
	out.Init(buf, 0, 8, 2, ' ', 0)

	f(out)

	out.Flush()

	return buf.String()
}

func newCmdDescribe(rootCmdOptions *RootCmdOptions) *cobra.Command {
	cmd := cobra.Command{
		Use:   "describe",
		Short: "Describe a resource",
		Long:  `Describe a Camel K resource.`,
	}

	cmd.AddCommand(newDescribeKitCmd(rootCmdOptions))
	cmd.AddCommand(newDescribeIntegrationCmd(rootCmdOptions))
	cmd.AddCommand(newDescribePlatformCmd(rootCmdOptions))

	return &cmd
}
