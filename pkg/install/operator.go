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
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func InstallOperator(namespace string) error {
	return installResources(namespace,
		"operator-service-account.yaml",
		"operator-role-openshift.yaml", // TODO distinguish between Openshift and Kubernetes
		"operator-role-binding.yaml",
		"operator-deployment.yaml",
		"operator-service.yaml",
	)
}

func installResources(namespace string, names ...string) error {
	for _, name := range names {
		if err := installResource(namespace, name); err != nil {
			return err
		}
	}
	return nil
}

func installResource(namespace string, name string) error {
	obj, err := kubernetes.LoadResourceFromYaml(deploy.Resources[name])
	if err != nil {
		return err
	}

	if kObj, ok := obj.(metav1.Object); ok {
		kObj.SetNamespace(namespace)
	}

	err = sdk.Create(obj)
	if err != nil && errors.IsAlreadyExists(err) {
		// Don't recreate Service object
		if obj.GetObjectKind().GroupVersionKind().Kind == "Service" {
			return nil
		}
		return sdk.Update(obj)
	}
	return err
}
