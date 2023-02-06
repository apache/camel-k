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
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/assert"
)

func NoErrorAndNotEmptyBytes(t *testing.T, path string, callable func(path string) ([]byte, error)) {
	t.Helper()

	object, err := callable(path)

	assert.Nil(t, err)
	assert.NotEmpty(t, object)
}
func NoErrorAndNotEmptyString(t *testing.T, path string, callable func(path string) (string, error)) {
	t.Helper()

	object, err := callable(path)

	assert.Nil(t, err)
	assert.NotEmpty(t, object)
}

func NoErrorAndContains(t *testing.T, path string, contains string, callable func(path string) ([]string, error)) {
	t.Helper()

	elements, err := callable(path)

	assert.Nil(t, err)
	assert.Contains(t, elements, contains)
}
func NoErrorAndNotContains(t *testing.T, path string, contains string, callable func(path string) ([]string, error)) {
	t.Helper()

	elements, err := callable(path)

	assert.Nil(t, err)
	assert.NotContains(t, elements, contains)
}
func NoErrorAndEmpty(t *testing.T, path string, callable func(path string) ([]string, error)) {
	t.Helper()

	elements, err := callable(path)

	assert.Nil(t, err)
	assert.Empty(t, elements)
}

func ErrorBytes(t *testing.T, path string, callable func(path string) ([]byte, error)) {
	t.Helper()

	_, err := callable(path)
	assert.NotNil(t, err)
}
func ErrorString(t *testing.T, path string, callable func(path string) (string, error)) {

	t.Helper()

	_, err := callable(path)
	assert.NotNil(t, err)
}

func TestGetResource(t *testing.T) {
	NoErrorAndNotEmptyBytes(t, "manager/operator-service-account.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/manager/operator-service-account.yaml", Resource)
	NoErrorAndNotEmptyString(t, "manager/operator-service-account.yaml", ResourceAsString)
	NoErrorAndNotEmptyString(t, "/manager/operator-service-account.yaml", ResourceAsString)
	NoErrorAndContains(t, "/manager", "/manager/operator-service-account.yaml", Resources)
}

func TestGetNoResource(t *testing.T) {
	ErrorBytes(t, "manager/operator-service-account.json", Resource)
	ErrorBytes(t, "/manager/operator-service-account.json", Resource)
	ErrorString(t, "manager/operator-service-account.json", ResourceAsString)
	ErrorString(t, "/manager/operator-service-account.json", ResourceAsString)
	NoErrorAndNotContains(t, "/", "/manager/operator-service-account.json", Resources)
}

func TestResources(t *testing.T) {
	NoErrorAndContains(t, "/manager", "/manager/operator-service-account.yaml", Resources)
	NoErrorAndContains(t, "/manager/", "/manager/operator-service-account.yaml", Resources)
	NoErrorAndNotContains(t, "/manager/", "kustomize.yaml", Resources)
	NoErrorAndEmpty(t, "/dirnotexist", Resources)

	items, err := Resources("/")
	assert.Nil(t, err)

	for _, res := range items {
		if strings.Contains(res, "java.tmpl") {
			assert.Fail(t, "Resources should not return nested files")
		}
		if strings.Contains(res, "templates") {
			assert.Fail(t, "Resources should not return nested dirs")
		}
	}

	NoErrorAndContains(t, "/templates", "/templates/java.tmpl", Resources)
}

func TestResourcesWithPrefix(t *testing.T) {
	NoErrorAndContains(t, "/manager/", "/manager/operator-service-account.yaml", WithPrefix)
	NoErrorAndContains(t, "/manager/op", "/manager/operator-service-account.yaml", WithPrefix)
	NoErrorAndContains(t, "/manager/operator-service-account", "/manager/operator-service-account.yaml", WithPrefix)
	NoErrorAndContains(t, "/traits", "/traits.yaml", WithPrefix)

	// directory needs the slash on the end
	NoErrorAndNotContains(t, "/manager", "/manager/operator-service-account.yaml", WithPrefix)

	// need to get to at least the same directory as the required files
	NoErrorAndNotContains(t, "/", "/manager/operator-service-account.yaml", WithPrefix)
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

	data, err := TemplateResource(fmt.Sprintf("/addons/master/%s", fname), templateData)
	assert.NoError(t, err)

	jsonSrc, err := yaml.ToJSON([]byte(data))
	assert.NoError(t, err)

	uns := unstructured.Unstructured{}
	err = uns.UnmarshalJSON(jsonSrc)
	assert.NoError(t, err)

	assert.Equal(t, uns.GetName(), name)
	assert.Equal(t, uns.GetNamespace(), ns)
}

func TestCRDResources(t *testing.T) {
	NoErrorAndNotEmptyBytes(t, "/crd/bases/camel.apache.org_builds.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/crd/bases/camel.apache.org_camelcatalogs.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/crd/bases/camel.apache.org_integrationkits.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/crd/bases/camel.apache.org_integrationplatforms.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/crd/bases/camel.apache.org_integrations.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/crd/bases/camel.apache.org_kameletbindings.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/crd/bases/camel.apache.org_kamelets.yaml", Resource)
}
