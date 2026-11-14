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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
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
	NoErrorAndContains(t, "/config/manager", "config/manager/operator-service-account.yaml", Resources)
}

func TestGetNoResource(t *testing.T) {
	ErrorBytes(t, "config/manager/operator-service-account.json", Resource)
	ErrorBytes(t, "/config/manager/operator-service-account.json", Resource)
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

func TestCRDResources(t *testing.T) {
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_builds.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_camelcatalogs.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrationkits.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrationplatforms.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrationprofiles.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_integrations.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_kamelets.yaml", Resource)
	NoErrorAndNotEmptyBytes(t, "/config/crd/bases/camel.apache.org_pipes.yaml", Resource)
}

// StorageClass is a cluster scoped resource: a namespaced Role can never grant access to it,
// so the permission must be granted through a dedicated ClusterRole in every installation mode.
func TestStorageClassesRBACIsClusterScoped(t *testing.T) {
	hasStorageClasses := func(rules []rbacv1.PolicyRule) bool {
		for _, rule := range rules {
			if slices.Contains(rule.Resources, "storageclasses") {
				return true
			}
		}

		return false
	}

	var role rbacv1.Role
	data, err := Resource("config/rbac/namespaced/operator-role.yaml")
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &role))
	assert.False(t, hasStorageClasses(role.Rules), "namespaced operator Role cannot grant access to cluster scoped storageclasses")

	var descopedRole rbacv1.ClusterRole
	data, err = Resource("config/rbac/descoped/operator-cluster-role.yaml")
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &descopedRole))
	assert.False(t, hasStorageClasses(descopedRole.Rules), "storageclasses must be granted by the common ClusterRole only")

	var clusterRole rbacv1.ClusterRole
	data, err = Resource("config/rbac/get-storageclasses-cluster-role.yaml")
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &clusterRole))
	require.True(t, hasStorageClasses(clusterRole.Rules))
	assert.Equal(t, []string{"storage.k8s.io"}, clusterRole.Rules[0].APIGroups)
	assert.ElementsMatch(t, []string{"get", "list"}, clusterRole.Rules[0].Verbs)

	var binding rbacv1.ClusterRoleBinding
	data, err = Resource("config/rbac/get-storageclasses-cluster-role-binding.yaml")
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(data, &binding))
	assert.Equal(t, clusterRole.Name, binding.RoleRef.Name)
	assert.Equal(t, "ClusterRole", binding.RoleRef.Kind)
	require.Len(t, binding.Subjects, 1)
	assert.Equal(t, "camel-k-operator", binding.Subjects[0].Name)

	// the common RBAC kustomization must ship them, so they apply to every installation mode
	kustomization, err := Resource("config/rbac/kustomization.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(kustomization), "get-storageclasses-cluster-role.yaml")
	assert.Contains(t, string(kustomization), "get-storageclasses-cluster-role-binding.yaml")
}
