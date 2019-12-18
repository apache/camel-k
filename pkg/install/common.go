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
	"strings"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/openshift"
	kube "k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceCustomizer can be used to inject code that changes the objects before they are created
type ResourceCustomizer func(object runtime.Object) runtime.Object

// IdentityResourceCustomizer is a ResourceCustomizer that does nothing
var IdentityResourceCustomizer = func(object runtime.Object) runtime.Object {
	return object
}

// Resources installs named resources from the project resource directory
func Resources(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, names ...string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, customizer, names...)
}

// ResourcesOrCollect --
func ResourcesOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, customizer ResourceCustomizer, names ...string) error {
	for _, name := range names {
		if err := ResourceOrCollect(ctx, c, namespace, collection, customizer, name); err != nil {
			return err
		}
	}
	return nil
}

// Resource installs a single named resource from the project resource directory
func Resource(ctx context.Context, c client.Client, namespace string, customizer ResourceCustomizer, name string) error {
	return ResourceOrCollect(ctx, c, namespace, nil, customizer, name)
}

// ResourceOrCollect --
func ResourceOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, customizer ResourceCustomizer, name string) error {
	obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), deploy.Resources[name])
	if err != nil {
		return err
	}

	return RuntimeObjectOrCollect(ctx, c, namespace, collection, customizer(obj))
}

// RuntimeObject installs a single runtime object
func RuntimeObject(ctx context.Context, c client.Client, namespace string, obj runtime.Object) error {
	return RuntimeObjectOrCollect(ctx, c, namespace, nil, obj)
}

// RuntimeObjectOrCollect --
func RuntimeObjectOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, obj runtime.Object) error {
	if collection != nil {
		// Adding to the collection before setting the namespace
		collection.Add(obj)
		return nil
	}

	if metaObject, ok := obj.(metav1.Object); ok {
		metaObject.SetNamespace(namespace)
	}

	err := c.Create(ctx, obj)
	if err != nil && errors.IsAlreadyExists(err) {
		// Don't recreate Service object
		if obj.GetObjectKind().GroupVersionKind().Kind == "Service" {
			return nil
		}
		// Don't recreate integration kits, platforms, etc
		if obj.GetObjectKind().GroupVersionKind().Kind == v1.IntegrationKindKind {
			return nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == v1.IntegrationPlatformKind {
			return nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == v1.CamelCatalogKind {
			return nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == v1.BuildKind {
			return nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == "PersistentVolumeClaim" {
			return nil
		}
		return c.Update(ctx, obj)
	}
	return err
}

func isOpenShift(c kube.Interface, clusterType string) (bool, error) {
	var res bool
	var err error
	if clusterType != "" {
		res = strings.EqualFold(clusterType, string(v1.IntegrationPlatformClusterOpenShift))
	} else {
		res, err = openshift.IsOpenShift(c)
		if err != nil {
			return false, err
		}
	}
	return res, nil
}
