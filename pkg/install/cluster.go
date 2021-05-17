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
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/resources"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func SetupClusterWideResourcesOrCollect(ctx context.Context, clientProvider client.Provider, collection *kubernetes.Collection, clusterType string, force bool) error {
	// Get a client to install the CRD
	c, err := clientProvider.Get()
	if err != nil {
		return err
	}

	isApiExtensionsV1 := true
	_, err = c.Discovery().ServerResourcesForGroupVersion("apiextensions.k8s.io/v1")
	if err != nil && k8serrors.IsNotFound(err) {
		isApiExtensionsV1 = false
	} else if err != nil {
		return err
	}

	// Convert the CRD to apiextensions.k8s.io/v1beta1 in case v1 is not available.
	// This is mainly required to support OpenShift 3, and older versions of Kubernetes.
	// It can be removed as soon as these versions are not supported anymore.
	err = apiextensionsv1.AddToScheme(c.GetScheme())
	if err != nil {
		return err
	}
	if !isApiExtensionsV1 {
		err = apiextensionsv1beta1.AddToScheme(c.GetScheme())
		if err != nil {
			return err
		}
	}
	downgradeToCRDv1beta1 := func(object ctrl.Object) ctrl.Object {
		// Remove default values in v1beta1 Integration and KameletBinding CRDs,
		removeDefaultFromCrd := func(crd *apiextensionsv1beta1.CustomResourceDefinition, property string) {
			defaultValue := apiextensionsv1beta1.JSONSchemaProps{
				Default: nil,
			}
			if crd.Name == "integrations.camel.apache.org" {
				crd.Spec.Validation.OpenAPIV3Schema.
					Properties["spec"].Properties["template"].Properties["spec"].Properties[property].Items.Schema.
					Properties["ports"].Items.Schema.Properties["protocol"] = defaultValue
			}
			if crd.Name == "kameletbindings.camel.apache.org" {
				crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"].Properties["integration"].Properties["template"].
					Properties["spec"].Properties[property].Items.Schema.Properties["ports"].Items.Schema.
					Properties["protocol"] = defaultValue
			}
		}

		if !isApiExtensionsV1 {
			v1Crd := object.(*apiextensionsv1.CustomResourceDefinition)
			v1beta1Crd := &apiextensionsv1beta1.CustomResourceDefinition{}
			crd := &apiextensions.CustomResourceDefinition{}

			err := apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(v1Crd, crd, nil)
			if err != nil {
				return nil
			}

			err = apiextensionsv1beta1.Convert_apiextensions_CustomResourceDefinition_To_v1beta1_CustomResourceDefinition(crd, v1beta1Crd, nil)
			if err != nil {
				return nil
			}

			removeDefaultFromCrd(v1beta1Crd, "ephemeralContainers")
			removeDefaultFromCrd(v1beta1Crd, "containers")
			removeDefaultFromCrd(v1beta1Crd, "initContainers")

			return v1beta1Crd
		}
		return object
	}

	// Install CRD for Integration Platform (if needed)
	if err := installCRD(ctx, c, "IntegrationPlatform", "v1", "camel.apache.org_integrationplatforms.yaml", downgradeToCRDv1beta1, collection, force); err != nil {
		return err
	}

	// Install CRD for Integration Kit (if needed)
	if err := installCRD(ctx, c, "IntegrationKit", "v1", "camel.apache.org_integrationkits.yaml", downgradeToCRDv1beta1, collection, force); err != nil {
		return err
	}

	// Install CRD for Integration (if needed)
	if err := installCRD(ctx, c, "Integration", "v1", "camel.apache.org_integrations.yaml", downgradeToCRDv1beta1, collection, force); err != nil {
		return err
	}

	// Install CRD for Camel Catalog (if needed)
	if err := installCRD(ctx, c, "CamelCatalog", "v1", "camel.apache.org_camelcatalogs.yaml", downgradeToCRDv1beta1, collection, force); err != nil {
		return err
	}

	// Install CRD for Build (if needed)
	if err := installCRD(ctx, c, "Build", "v1", "camel.apache.org_builds.yaml", downgradeToCRDv1beta1, collection, force); err != nil {
		return err
	}

	// Install CRD for Kamelet (if needed)
	if err := installCRD(ctx, c, "Kamelet", "v1alpha1", "camel.apache.org_kamelets.yaml", downgradeToCRDv1beta1, collection, force); err != nil {
		return err
	}

	// Install CRD for KameletBinding (if needed)
	if err := installCRD(ctx, c, "KameletBinding", "v1alpha1", "camel.apache.org_kameletbindings.yaml", downgradeToCRDv1beta1, collection, force); err != nil {
		return err
	}

	// Don't wait if we're just collecting resources
	if collection == nil {
		// Wait for all CRDs to be installed before proceeding
		if err := WaitForAllCrdInstallation(ctx, clientProvider, 25*time.Second); err != nil {
			return err
		}
	}

	// Installing ClusterRoles
	ok, err := isClusterRoleInstalled(ctx, c, "camel-k:edit")
	if err != nil {
		return err
	}
	if !ok || collection != nil {
		err := installResource(ctx, c, collection, "/rbac/user-cluster-role.yaml")
		if err != nil {
			return err
		}
	}

	isOpenShift, err := isOpenShift(c, clusterType)
	if err != nil {
		return err
	}
	if isOpenShift {
		ok, err := isClusterRoleInstalled(ctx, c, "camel-k-operator-openshift")
		if err != nil {
			return err
		}
		if !ok || collection != nil {
			err := installResource(ctx, c, collection, "/rbac/operator-cluster-role-openshift.yaml")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func WaitForAllCrdInstallation(ctx context.Context, clientProvider client.Provider, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		var c client.Client
		var err error
		if c, err = clientProvider.Get(); err != nil {
			return err
		}
		var inst bool
		if inst, err = areAllCrdInstalled(ctx, c); err != nil {
			return err
		} else if inst {
			return nil
		}
		// Check after 2 seconds if not expired
		if time.Now().After(deadline) {
			return errors.New("cannot check CRD installation after " + strconv.FormatInt(timeout.Nanoseconds()/1000000000, 10) + " seconds")
		}
		time.Sleep(2 * time.Second)
	}
}

func areAllCrdInstalled(ctx context.Context, c client.Client) (bool, error) {
	if ok, err := isCrdInstalled(ctx, c, "IntegrationPlatform", "v1"); err != nil {
		return ok, err
	} else if !ok {
		return false, nil
	}
	if ok, err := isCrdInstalled(ctx, c, "IntegrationKit", "v1"); err != nil {
		return ok, err
	} else if !ok {
		return false, nil
	}
	if ok, err := isCrdInstalled(ctx, c, "Integration", "v1"); err != nil {
		return ok, err
	} else if !ok {
		return false, nil
	}
	if ok, err := isCrdInstalled(ctx, c, "CamelCatalog", "v1"); err != nil {
		return ok, err
	} else if !ok {
		return false, nil
	}
	if ok, err := isCrdInstalled(ctx, c, "Build", "v1"); err != nil {
		return ok, err
	} else if !ok {
		return false, nil
	}
	if ok, err := isCrdInstalled(ctx, c, "Kamelet", "v1alpha1"); err != nil {
		return ok, err
	} else if !ok {
		return false, nil
	}
	return isCrdInstalled(ctx, c, "KameletBinding", "v1alpha1")
}

func isCrdInstalled(ctx context.Context, c client.Client, kind string, version string) (bool, error) {
	lst, err := c.Discovery().ServerResourcesForGroupVersion(fmt.Sprintf("camel.apache.org/%s", version))
	if err != nil && k8serrors.IsNotFound(err) {
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

func installCRD(ctx context.Context, c client.Client, kind string, version string, resourceName string, converter ResourceCustomizer, collection *kubernetes.Collection, force bool) error {
	crd, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), resources.ResourceAsString("/crd/bases/"+resourceName))
	if err != nil {
		return err
	}

	crd = converter(crd)
	if crd == nil {
		// The conversion has failed
		return errors.New("cannot convert " + resourceName + " CRD to apiextensions.k8s.io/v1beta1")
	}

	if collection != nil {
		collection.Add(crd)
		return nil
	}

	installed, err := isCrdInstalled(ctx, c, kind, version)
	if err != nil {
		return err
	}
	if installed && !force {
		return nil
	}

	return kubernetes.ReplaceResource(ctx, c, crd)
}

func isClusterRoleInstalled(ctx context.Context, c client.Client, name string) (bool, error) {
	clusterRole := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return isResourceInstalled(ctx, c, &clusterRole)
}

func isResourceInstalled(ctx context.Context, c client.Client, object ctrl.Object) (bool, error) {
	err := c.Get(ctx, ctrl.ObjectKeyFromObject(object), object)
	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func installResource(ctx context.Context, c client.Client, collection *kubernetes.Collection, resource string) error {
	obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), resources.ResourceAsString(resource))
	if err != nil {
		return err
	}
	if collection != nil {
		collection.Add(obj)
		return nil
	}
	return c.Create(ctx, obj)
}
