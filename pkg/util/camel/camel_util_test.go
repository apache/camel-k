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
	"sort"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
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

func TestMissingMatch(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.0", Provider: v1.RuntimeProviderQuarkus}}},
	}

	c, err := findBestMatch(catalogs, v1.RuntimeSpec{Version: "1.0.1", Provider: v1.RuntimeProviderQuarkus})
	assert.Nil(t, err)
	assert.Nil(t, c)
}

func TestNewCatalogVersionCollection(t *testing.T) {
	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.0.0", Provider: v1.RuntimeProviderQuarkus}}},
	}

	versions := make([]CatalogVersion, 0, len(catalogs))
	rv, _ := semver.NewVersion(catalogs[0].Spec.Runtime.Version)
	versions = append(versions, CatalogVersion{
		RuntimeVersion: rv,
		Catalog:        &catalogs[0],
	})
	expected := CatalogVersionCollection(versions)
	sort.Sort(sort.Reverse(expected))

	c := newCatalogVersionCollection(catalogs)

	assert.Equal(t, expected, c)

}
func TestIncorrectConstraint(t *testing.T) {
	rc := newSemVerConstraint("1.A.0")
	assert.Nil(t, rc)

	catalogs := []v1.CamelCatalog{
		{Spec: v1.CamelCatalogSpec{Runtime: v1.RuntimeSpec{Version: "1.A.0", Provider: v1.RuntimeProviderQuarkus}}},
	}
	c, err := findBestMatch(catalogs, v1.RuntimeSpec{Version: "1.0.0", Provider: v1.RuntimeProviderQuarkus})
	assert.Nil(t, err)
	assert.Nil(t, c)
}

func TestGetDependency(t *testing.T) {
	artifact := v1.CamelArtifact{}
	artifact.ArtifactID = "camel-quarkus-"
	provider := v1.RuntimeProviderQuarkus
	assert.Equal(t, "camel:", getDependency(artifact, provider))
	provider = v1.RuntimeProvider("notquarkus")
	artifact.ArtifactID = "camel-"
	assert.Equal(t, "camel:", getDependency(artifact, provider))
}
