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
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/apache/camel-k/v2/pkg/util/boolean"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateCatalog(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Status.Build.Timeout = &metav1.Duration{
		Duration: 5 * time.Minute,
	}
	c, err := test.NewFakeClient()
	require.NoError(t, err)
	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}
	if strings.Contains(defaults.DefaultRuntimeVersion, "SNAPSHOT") {
		maven.DefaultMavenRepositories += ",https://repository.apache.org/content/repositories/snapshots-group@snapshots@id=apache-snapshots"
	}
	catalog, err := CreateCatalog(
		context.TODO(),
		c,
		"",
		&ip,
		v1.RuntimeSpec{Provider: v1.RuntimeProviderQuarkus, Version: defaults.DefaultRuntimeVersion})
	require.NoError(t, err)
	assert.NotNil(t, catalog)
	assert.Equal(t, defaults.DefaultRuntimeVersion, catalog.Runtime.Version)
	assert.Equal(t, v1.RuntimeProviderQuarkus, catalog.Runtime.Provider)
	assert.NotEmpty(t, catalog.Runtime.Capabilities)

	camelCatalog := v1.CamelCatalog{
		Spec:   catalog.CamelCatalogSpec,
		Status: catalog.CamelCatalogStatus,
	}

	rtCat := NewRuntimeCatalog(camelCatalog)
	assert.NotNil(t, rtCat.Runtime.Capabilities["knative"])
}
