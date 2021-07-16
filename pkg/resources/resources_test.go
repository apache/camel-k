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

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestGetResource(t *testing.T) {
	assert.NotEmpty(t, Resource("manager/operator-service-account.yaml"))
	assert.NotEmpty(t, Resource("/manager/operator-service-account.yaml"))
	assert.NotEmpty(t, ResourceAsString("manager/operator-service-account.yaml"))
	assert.NotEmpty(t, ResourceAsString("/manager/operator-service-account.yaml"))
	assert.Contains(t, Resources("/manager"), "/manager/operator-service-account.yaml")
}

func TestGetNoResource(t *testing.T) {
	assert.Empty(t, Resource("manager/operator-service-account.json"))
	assert.Empty(t, Resource("/manager/operator-service-account.json"))
	assert.Empty(t, ResourceAsString("manager/operator-service-account.json"))
	assert.Empty(t, ResourceAsString("/manager/operator-service-account.json"))
	assert.NotContains(t, Resources("/"), "/manager/operator-service-account.json")
}

func TestResources(t *testing.T) {
	assert.Contains(t, Resources("/manager"), "/manager/operator-service-account.yaml")
	assert.Contains(t, Resources("/manager/"), "/manager/operator-service-account.yaml")
	assert.NotContains(t, Resources("/manager"), "kustomize.yaml")
	assert.Empty(t, Resources("/dirnotexist"))

	for _, res := range Resources("/") {
		if strings.Contains(res, "java.tmpl") {
			assert.Fail(t, "Resources should not return nested files")
		}
		if strings.Contains(res, "templates") {
			assert.Fail(t, "Resources should not return nested dirs")
		}
	}
	assert.Contains(t, Resources("/templates"), "/templates/java.tmpl")
}

func TestResourcesWithPrefix(t *testing.T) {
	assert.Contains(t, ResourcesWithPrefix("/manager/"), "/manager/operator-service-account.yaml")
	assert.Contains(t, ResourcesWithPrefix("/manager/op"), "/manager/operator-service-account.yaml")
	assert.Contains(t, ResourcesWithPrefix("/manager/operator-service-account"), "/manager/operator-service-account.yaml")

	assert.Contains(t, ResourcesWithPrefix("/traits"), "/traits.yaml")

	// directory needs the slash on the end
	assert.NotContains(t, ResourcesWithPrefix("/manager"), "/manager/operator-service-account.yaml")
	// need to get to at least the same directory as the required files
	assert.NotContains(t, ResourcesWithPrefix("/"), "/manager/operator-service-account.yaml")
}

func TestTemplateResource(t *testing.T) {
	fname := "master-role-lease.tmpl"
	name := "myintegration-master"
	ns := "test-nm"

	var templateData = struct {
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
	assert.NotEmpty(t, Resource("/crd/bases/camel.apache.org_builds.yaml"))
	assert.NotEmpty(t, Resource("/crd/bases/camel.apache.org_camelcatalogs.yaml"))
	assert.NotEmpty(t, Resource("/crd/bases/camel.apache.org_integrationkits.yaml"))
	assert.NotEmpty(t, Resource("/crd/bases/camel.apache.org_integrationplatforms.yaml"))
	assert.NotEmpty(t, Resource("/crd/bases/camel.apache.org_integrations.yaml"))
	assert.NotEmpty(t, Resource("/crd/bases/camel.apache.org_kameletbindings.yaml"))
	assert.NotEmpty(t, Resource("/crd/bases/camel.apache.org_kamelets.yaml"))
}
