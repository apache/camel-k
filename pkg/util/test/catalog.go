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

package test

import (
	"fmt"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	yaml "gopkg.in/yaml.v2"
)

// DefaultCatalog --
func DefaultCatalog() (*camel.RuntimeCatalog, error) {
	data, ok := deploy.Resources["camel-catalog-"+defaults.CamelVersion+".yaml"]
	if !ok {
		return nil, fmt.Errorf("unable to find default catalog from embedded resources")
	}

	var c v1alpha1.CamelCatalog
	if err := yaml.Unmarshal([]byte(data), &c); err != nil {
		return nil, err
	}

	return camel.NewRuntimeCatalog(c.Spec), nil
}
