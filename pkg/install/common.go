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
	"time"

	"github.com/sirupsen/logrus"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources installs named resources from the project resource directory
func Resources(ctx context.Context, c client.Client, namespace string, names ...string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, names...)
}

// ResourcesOrCollect --
func ResourcesOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, names ...string) error {
	for _, name := range names {
		obj, err := ResourceOrCollect(ctx, c, namespace, collection, name)
		if err != nil {
			return err
		}

		if ictx, ok := obj.(*v1alpha1.IntegrationContext); ok {
			for {
				key := k8sclient.ObjectKey{
					Name:      ictx.Name,
					Namespace: ictx.Namespace,
				}

				if err := c.Get(ctx, key, obj); err != nil {
					// ignore and go ahead
					break
				}

				if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseReady {
					break
				}
				if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseError {
					break
				}

				logrus.Infof("Waiting for IntegrationContext %s to be ready ...", ictx.Name)
				time.Sleep(1 * time.Second)
			}
		}
	}
	return nil
}

// Resource installs a single named resource from the project resource directory
func Resource(ctx context.Context, c client.Client, namespace string, name string) (runtime.Object, error) {
	return ResourceOrCollect(ctx, c, namespace, nil, name)
}

// ResourceOrCollect --
func ResourceOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, name string) (runtime.Object, error) {
	obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), deploy.Resources[name])
	if err != nil {
		return nil, err
	}

	return RuntimeObjectOrCollect(ctx, c, namespace, collection, obj)
}

// RuntimeObject installs a single runtime object
func RuntimeObject(ctx context.Context, c client.Client, namespace string, obj runtime.Object) (runtime.Object, error) {
	return RuntimeObjectOrCollect(ctx, c, namespace, nil, obj)
}

// RuntimeObjectOrCollect --
func RuntimeObjectOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, obj runtime.Object) (runtime.Object, error) {
	if collection != nil {
		// Adding to the collection before setting the namespace
		collection.Add(obj)
		return nil, nil
	}

	if metaObject, ok := obj.(metav1.Object); ok {
		metaObject.SetNamespace(namespace)
	}

	err := c.Create(ctx, obj)
	if err != nil && errors.IsAlreadyExists(err) {
		// Don't recreate Service object
		if obj.GetObjectKind().GroupVersionKind().Kind == "Service" {
			return obj, nil
		}
		// Don't recreate integration contexts or platforms
		if obj.GetObjectKind().GroupVersionKind().Kind == v1alpha1.IntegrationContextKind {
			return obj, nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == v1alpha1.IntegrationPlatformKind {
			return obj, nil
		}
		if obj.GetObjectKind().GroupVersionKind().Kind == "PersistentVolumeClaim" {
			return obj, nil
		}
		return obj, c.Update(ctx, obj)
	}
	return obj, err
}
