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

package trait

import (
	"fmt"

	"github.com/apache/camel-k/pkg/util/camel"
)

type camelTrait struct {
	BaseTrait `property:",squash"`
	Version   string `property:"version"`
}

func newCamelTrait() *camelTrait {
	return &camelTrait{
		BaseTrait: newBaseTrait("camel"),
	}
}

func (t *camelTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return true, nil
}

func (t *camelTrait) Apply(e *Environment) error {
	if e.Integration != nil {
		if e.CamelCatalog == nil {
			version := e.DetermineCamelVersion()

			if t.Version != "" {
				version = t.Version
			}

			c, err := camel.Catalog(e.C, e.Client, e.Integration.Namespace, version)
			if err != nil {
				return err
			}
			if c == nil {
				return fmt.Errorf("unable to find catalog for: %s", version)
			}

			e.CamelCatalog = c
		}

		e.Integration.Status.CamelVersion = e.CamelCatalog.Version
	}

	if e.IntegrationContext != nil {
		if e.CamelCatalog == nil {
			version := e.DetermineCamelVersion()

			if t.Version != "" {
				version = t.Version
			}

			c, err := camel.Catalog(e.C, e.Client, e.IntegrationContext.Namespace, version)
			if err != nil {
				return err
			}
			if c == nil {
				return fmt.Errorf("unable to find catalog for: %s", version)
			}

			e.CamelCatalog = c
		}

		e.IntegrationContext.Status.CamelVersion = e.CamelCatalog.Version
	}

	return nil
}
