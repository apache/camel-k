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

package resources

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func NoErrorAndNotEmptyBytes(t *testing.T, path string, callable func(path string) ([]byte, error)) {
	t.Helper()

	object, err := callable(path)

	require.NoError(t, err)
	assert.NotEmpty(t, object)
}
func NoErrorAndNotEmptyString(t *testing.T, path string, callable func(path string) (string, error)) {
	t.Helper()

	object, err := callable(path)

	require.NoError(t, err)
	assert.NotEmpty(t, object)
}

func NoErrorAndContains(t *testing.T, path string, contains string, callable func(path string) ([]string, error)) {
	t.Helper()

	elements, err := callable(path)

	require.NoError(t, err)
	assert.Contains(t, elements, contains)
}
func NoErrorAndNotContains(t *testing.T, path string, contains string, callable func(path string) ([]string, error)) {
	t.Helper()

	elements, err := callable(path)

	require.NoError(t, err)
	assert.NotContains(t, elements, contains)
}
func NoErrorAndEmpty(t *testing.T, path string, callable func(path string) ([]string, error)) {
	t.Helper()

	elements, err := callable(path)

	require.NoError(t, err)
	assert.Empty(t, elements)
}

func ErrorBytes(t *testing.T, path string, callable func(path string) ([]byte, error)) {
	t.Helper()

	_, err := callable(path)
	require.Error(t, err)
}
func ErrorString(t *testing.T, path string, callable func(path string) (string, error)) {

	t.Helper()

	_, err := callable(path)
	require.Error(t, err)
}

func TestGetResource(t *testing.T) {
	NoErrorAndNotEmptyBytes(t, "config/manager/operator-service-account.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/manager/operator-service-account.yaml", Resource)
	NoErrorAndNotEmptyString(t, "config/manager/operator-service-account.yaml", ResourceAsString)
	NoErrorAndNotEmptyString(t, "/config/manager/operator-service-account.yaml", ResourceAsString)
	NoErrorAndContains(t, "/config/manager", "config/manager/operator-service-account.yaml", Resources)
}

func TestGetNoResource(t *testing.T) {
	ErrorBytes(t, "config/manager/operator-service-account.json", Resource)
	ErrorBytes(t, "/config/manager/operator-service-account.json", Resource)
	ErrorString(t, "config/manager/operator-service-account.json", ResourceAsString)
	ErrorString(t, "/config/manager/operator-service-account.json", ResourceAsString)
	NoErrorAndNotContains(t, "/config/", "config/manager/operator-service-account.json", Resources)
}

func TestResources(t *testing.T) {
	NoErrorAndContains(t, "/config/manager", "config/manager/operator-service-account.yaml", Resources)
	NoErrorAndContains(t, "/config/manager/", "config/manager/operator-service-account.yaml", Resources)
	NoErrorAndNotContains(t, "config/manager/", "config/kustomize.yaml", Resources)
	NoErrorAndEmpty(t, "config/dirnotexist", Resources)

	_, err := Resources("/")
	require.NoError(t, err)
}

func TestResourcesWithPrefix(t *testing.T) {
	NoErrorAndContains(t, "/config/manager/", "config/manager/operator-service-account.yaml", WithPrefix)
	NoErrorAndContains(t, "/config/manager/op", "config/manager/operator-service-account.yaml", WithPrefix)
	NoErrorAndContains(t, "/config/manager/operator-service-account", "config/manager/operator-service-account.yaml", WithPrefix)

	// directory needs the slash on the end
	NoErrorAndNotContains(t, "/config/manager", "config/manager/operator-service-account.yaml", WithPrefix)

	// need to get to at least the same directory as the required files
	NoErrorAndNotContains(t, "/", "config/manager/operator-service-account.yaml", WithPrefix)
}

func TestTemplateResource(t *testing.T) {
	fname := "master-role-lease.tmpl"
	name := "myintegration-master"
	ns := "test-nm"

	templateData := struct {
		Namespace      string
		Name           string
		ServiceAccount string
	}{
		Namespace:      ns,
		Name:           name,
		ServiceAccount: "default",
	}

	data, err := TemplateResource(fmt.Sprintf("/resources/addons/master/%s", fname), templateData)
	require.NoError(t, err)

	jsonSrc, err := yaml.ToJSON([]byte(data))
	require.NoError(t, err)

	uns := unstructured.Unstructured{}
	err = uns.UnmarshalJSON(jsonSrc)
	require.NoError(t, err)

	assert.Equal(t, uns.GetName(), name)
	assert.Equal(t, uns.GetNamespace(), ns)
}

func TestCRDResources(t *testing.T) {
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_builds.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_camelcatalogs.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrationkits.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrationplatforms.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrationprofiles.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrations.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_kamelets.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_kameletbindings.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_pipes.yaml", Resource)
}
