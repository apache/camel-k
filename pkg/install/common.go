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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Resources installs named resources from the project resource directory
func Resources(namespace string, names ...string) error {
	return ResourcesOrCollect(namespace, nil, names...)
}

func ResourcesOrCollect(namespace string, collection *kubernetes.Collection, names ...string) error {
	for _, name := range names {
		if err := ResourceOrCollect(namespace, collection, name); err != nil {
			return err
		}
	}
	return nil
}

// Resource installs a single named resource from the project resource directory
func Resource(namespace string, name string) error {
	return ResourceOrCollect(namespace, nil, name)
}

func ResourceOrCollect(namespace string, collection *kubernetes.Collection, name string) error {
	obj, err := kubernetes.LoadResourceFromYaml(deploy.Resources[name])
	if err != nil {
		return err
	}

	return RuntimeObjectOrCollect(namespace, collection, obj)
}

// RuntimeObject installs a single runtime object
func RuntimeObject(namespace string, obj runtime.Object) error {
	return RuntimeObjectOrCollect(namespace, nil, obj)
}

func RuntimeObjectOrCollect(namespace string, collection *kubernetes.Collection, obj runtime.Object) error {
	if collection != nil {
		// Adding to the collection before setting the namespace
		collection.Add(obj)
		return nil
	}

	if metaObject, ok := obj.(metav1.Object); ok {
		metaObject.SetNamespace(namespace)
	}

	err := sdk.Create(obj)
	if err != nil && errors.IsAlreadyExists(err) {
		// Don't recreate Service object
		if obj.GetObjectKind().GroupVersionKind().Kind == "Service" {
			return nil
		}
		// Don't recreate integration contexts or platforms
		if obj.GetObjectKind().GroupVersionKind().Kind == v1alpha1.IntegrationContextKind {
			return nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == v1alpha1.IntegrationPlatformKind {
			return nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == "PersistentVolumeClaim" {
			return nil
		}
		return sdk.Update(obj)
	}
	return err
}
