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
	"strings"
	"time"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/indentedwriter"
)

func describeObjectMeta(w *indentedwriter.Writer, om metav1.ObjectMeta) {
	w.Writef(0, "Name:\t%s\n", om.Name)
	w.Writef(0, "Namespace:\t%s\n", om.Namespace)

	if len(om.GetLabels()) > 0 {
		w.Writef(0, "Labels:")
		for k, v := range om.Labels {
			w.Writef(0, "\t%s=%s\n", k, strings.TrimSpace(v))
		}
	}

	if len(om.GetAnnotations()) > 0 {
		w.Writef(0, "Annotations:")
		for k, v := range om.Annotations {
			w.Writef(0, "\t%s=%s\n", k, strings.TrimSpace(v))
		}
	}

	w.Writef(0, "Creation Timestamp:\t%s\n", om.CreationTimestamp.Format(time.RFC1123Z))
}

func describeTraits(w *indentedwriter.Writer, traits interface{}) error {
	traitsMap, err := trait.ToMap(traits)
	if err != nil {
		return err
	}

	if len(traitsMap) > 0 {
		w.Writef(0, "Traits:\n")

		for id, trait := range traitsMap {
			w.Writef(1, "%s:\n", strings.Title(id))
			// TODO: print the whole TraitSpec as Yaml
			for k, v := range trait {
				w.Writef(2, "%s:\t%v\n", strings.Title(k), v)
			}
		}
	}
	return nil
}

func newCmdDescribe(rootCmdOptions *RootCmdOptions) *cobra.Command {
	cmd := cobra.Command{
		Use:   "describe",
		Short: "Describe a resource",
		Long:  `Describe a Camel K resource.`,
	}

	cmd.AddCommand(cmdOnly(newDescribeKitCmd(rootCmdOptions)))
	cmd.AddCommand(cmdOnly(newDescribeIntegrationCmd(rootCmdOptions)))
	cmd.AddCommand(cmdOnly(newDescribePlatformCmd(rootCmdOptions)))
	cmd.AddCommand(cmdOnly(newDescribeKameletCmd(rootCmdOptions)))

	return &cmd
}
