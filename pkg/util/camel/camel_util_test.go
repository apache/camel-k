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

func TestFindBestMatch_Camel(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "~2.23.x", "1.0.0", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1", c.Version)
}

func TestFindBestMatch_Runtime(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.1"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "2.23.0", "~1.0.x", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.0", c.Version)
	assert.Equal(t, "1.0.1", c.RuntimeVersion)
}

func TestFindBestMatch(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.1"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "~2.23.x", "~1.0.x", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1", c.Version)
	assert.Equal(t, "1.0.1", c.RuntimeVersion)
}

func TestFindExactSemVerMatch_Camel(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "2.23.0", "1.0.0", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.0", c.Version)
}

func TestFindExactSemVerMatch_Runtime(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.1"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "2.23.0", "1.0.0", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.0", c.Version)
	assert.Equal(t, "1.0.0", c.RuntimeVersion)
}

func TestFindExactMatch_Camel(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1-tag-00001", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1-tag-00002", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "2.23.1-tag-00001", "1.0.0", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1-tag-00001", c.Version)
}

func TestFindExactMatch_Runtime(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1-tag-00001", RuntimeVersion: "1.0.1"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1-tag-00002", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "2.23.1-tag-00001", "1.0.1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1-tag-00001", c.Version)
	assert.Equal(t, "1.0.1", c.RuntimeVersion)
}

func TestFindRangeMatch_Camel(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.2", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, ">= 2.23.0, < 2.23.2", "1.0.0", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1", c.Version)
}

func TestFindRangeMatch_Runtime(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.2", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.2"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, "2.23.0", "> 1.0.1, < 1.0.3", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.0", c.Version)
	assert.Equal(t, "1.0.2", c.RuntimeVersion)
}

func TestFindRangeMatch(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.2", RuntimeVersion: "1.0.0"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.0", RuntimeVersion: "1.0.2"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.23.1", RuntimeVersion: "1.0.2"}},
		{Spec: v1.CamelCatalogSpec{Version: "2.22.1", RuntimeVersion: "1.0.0"}},
	}

	c, err := findBestMatch(catalogs, ">= 2.23.0, < 2.23.2", "> 1.0.1, < 1.0.3", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "2.23.1", c.Version)
	assert.Equal(t, "1.0.2", c.RuntimeVersion)
}
