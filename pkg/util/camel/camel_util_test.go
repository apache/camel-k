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

package camel

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestFindBestMatch(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.0", Provider: v1.RuntimeProviderQuarkus}}},
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.1", Provider: v1.RuntimeProviderQuarkus}}},
	}

	c, err := findBestMatch(catalogs, v1.RuntimeSpec{Version: "~1.0.x", Provider: v1.RuntimeProviderQuarkus})
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "1.0.1", c.Runtime.Version)
	assert.Equal(t, v1.RuntimeProviderQuarkus, c.Runtime.Provider)
}

func TestFindExactSemVerMatch(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.0", Provider: v1.RuntimeProviderQuarkus}}},
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.1", Provider: v1.RuntimeProviderQuarkus}}},
	}

	c, err := findBestMatch(catalogs, v1.RuntimeSpec{Version: "1.0.0", Provider: v1.RuntimeProviderQuarkus})
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "1.0.0", c.Runtime.Version)
	assert.Equal(t, v1.RuntimeProviderQuarkus, c.Runtime.Provider)
}

func TestFindRangeMatch(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.0", Provider: v1.RuntimeProviderQuarkus}}},
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.1", Provider: v1.RuntimeProviderQuarkus}}},
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.2", Provider: v1.RuntimeProviderQuarkus}}},
	}

	c, err := findBestMatch(catalogs, v1.RuntimeSpec{Version: "> 1.0.1, < 1.0.3", Provider: v1.RuntimeProviderQuarkus})
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "1.0.2", c.Runtime.Version)
	assert.Equal(t, v1.RuntimeProviderQuarkus, c.Runtime.Provider)
}
