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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/spf13/cobra"
)

func addDependency(cmd *cobra.Command, it *v1.Integration, dependency string, catalog *camel.RuntimeCatalog) {
	normalized := camel.NormalizeDependency(dependency)
	camel.ValidateDependency(catalog, normalized, cmd.ErrOrStderr())
	it.Spec.AddDependency(normalized)
}

func createCamelCatalog() (*camel.RuntimeCatalog, error) {
	// Attempt to reuse existing Camel catalog if one is present
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

func extractTraitNames(traitProps []string) []string {
	traitNameProps := make([]string, len(traitProps))
	for i, tp := range traitProps {
		splits := strings.Split(tp, ".")
		traitNameProps[i] = splits[0]
	}

	return traitNameProps
}
