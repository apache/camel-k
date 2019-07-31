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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func TestFindBestMatch(t *testing.T) {
	catalogs := []v1alpha1.CamelCatalog{
		{
			Spec: v1alpha1.CamelCatalogSpec{Version: "2.23.0"},
		},
		{
			Spec: v1alpha1.CamelCatalogSpec{Version: "2.23.1"},
		},
	}

	c, err := FindBestMatch("~2.23.x", catalogs)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1", c.Version)
}

func TestFindExactMatch(t *testing.T) {
	catalogs := []v1alpha1.CamelCatalog{
		{
			Spec: v1alpha1.CamelCatalogSpec{Version: "2.23.0"},
		},
		{
			Spec: v1alpha1.CamelCatalogSpec{Version: "2.23.1"},
		},
	}

	c, err := FindBestMatch("2.23.0", catalogs)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.0", c.Version)
}

func TestFindRangeMatch(t *testing.T) {
	catalogs := []v1alpha1.CamelCatalog{
		{
			Spec: v1alpha1.CamelCatalogSpec{Version: "2.23.0"},
		},
		{
			Spec: v1alpha1.CamelCatalogSpec{Version: "2.23.1"},
		},
		{
			Spec: v1alpha1.CamelCatalogSpec{Version: "2.23.2"},
		},
	}

	c, err := FindBestMatch(">= 2.23.0, < 2.23.2", catalogs)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1", c.Version)
}
