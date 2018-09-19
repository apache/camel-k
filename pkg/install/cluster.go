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

package install

import (
	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// SetupClusterwideResources --
func SetupClusterwideResources() error {

	// Install CRD for Integration Context (if needed)
	if err := installCRD("IntegrationContext", "crd-integration-context.yaml"); err != nil {
		return err
	}

	// Install CRD for Integration (if needed)
	if err := installCRD("Integration", "crd-integration.yaml"); err != nil {
		return err
	}

	// Installing ClusterRole
	clusterRoleInstalled, err := IsClusterRoleInstalled()
	if err != nil {
		return err
	}
	if !clusterRoleInstalled {
		err := installClusterRole()
		if err != nil {
			return err
		}
	}

	return nil
}

// IsCRDInstalled check if the given CRT kind is installed
func IsCRDInstalled(kind string) (bool, error) {
	lst, err := k8sclient.GetKubeClient().Discovery().ServerResourcesForGroupVersion("camel.apache.org/v1alpha1")
	if err != nil && errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	for _, res := range lst.APIResources {
		if res.Kind == kind {
			return true, nil
		}
	}
	return false, nil
}

func installCRD(kind string, resourceName string) error {
	// Installing Integration CRD
	installed, err := IsCRDInstalled(kind)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	crd := []byte(deploy.Resources[resourceName])
	crdJSON, err := yaml.ToJSON(crd)
	if err != nil {
		return err
	}
	restClient, err := customclient.GetClientFor("apiextensions.k8s.io", "v1beta1")
	if err != nil {
		return err
	}
	// Post using dynamic client
	result := restClient.
		Post().
		Body(crdJSON).
		Resource("customresourcedefinitions").
		Do()
	// Check result
	if result.Error() != nil && !errors.IsAlreadyExists(result.Error()) {
		return result.Error()
	}

	return nil
}

// IsClusterRoleInstalled check if cluster role camel-k:edit is installed
func IsClusterRoleInstalled() (bool, error) {
	clusterRole := v1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "camel-k:edit",
		},
	}
	err := sdk.Get(&clusterRole)
	if err != nil && errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func installClusterRole() error {
	obj, err := kubernetes.LoadResourceFromYaml(deploy.Resources["user-cluster-role.yaml"])
	if err != nil {
		return err
	}

	return sdk.Create(obj)
}
