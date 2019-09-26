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

package kubernetes

import (
	"context"

	"github.com/apache/camel-k/pkg/client"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReplaceResources allows to completely replace a list of resources on Kubernetes, taking care of immutable fields and resource versions
func ReplaceResources(ctx context.Context, c client.Client, objects []runtime.Object) error {
	for _, object := range objects {
		err := ReplaceResource(ctx, c, object)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReplaceResource allows to completely replace a resource on Kubernetes, taking care of immutable fields and resource versions
func ReplaceResource(ctx context.Context, c client.Client, res runtime.Object) error {
	err := c.Create(ctx, res)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		existing := res.DeepCopyObject()
		var key k8sclient.ObjectKey
		key, err = k8sclient.ObjectKeyFromObject(existing)
		if err != nil {
			return err
		}
		err = c.Get(ctx, key, existing)
		if err != nil {
			return err
		}
		mapRequiredMeta(existing, res)
		mapRequiredServiceData(existing, res)
		mapRequiredRouteData(existing, res)
		mapRequiredKnativeData(existing, res)
		err = c.Update(ctx, res)
	}
	if err != nil {
		return errors.Wrap(err, "could not create or replace "+findResourceDetails(res))
	}
	return nil
}

func mapRequiredMeta(from runtime.Object, to runtime.Object) {
	if fromC, ok := from.(metav1.Object); ok {
		if toC, ok := to.(metav1.Object); ok {
			toC.SetResourceVersion(fromC.GetResourceVersion())
		}
	}
}

func mapRequiredServiceData(from runtime.Object, to runtime.Object) {
	if fromC, ok := from.(*corev1.Service); ok {
		if toC, ok := to.(*corev1.Service); ok {
			toC.Spec.ClusterIP = fromC.Spec.ClusterIP
		}
	}
}

func mapRequiredRouteData(from runtime.Object, to runtime.Object) {
	if fromC, ok := from.(*routev1.Route); ok {
		if toC, ok := to.(*routev1.Route); ok {
			toC.Spec.Host = fromC.Spec.Host
		}
	}
}

func mapRequiredKnativeData(from runtime.Object, to runtime.Object) {
	if fromC, ok := from.(*messaging.Subscription); ok {
		if toC, ok := to.(*messaging.Subscription); ok {
			toC.Spec.DeprecatedGeneration = fromC.Spec.DeprecatedGeneration
		}
	}
}

func findResourceDetails(res runtime.Object) string {
	if res == nil {
		return "nil resource"
	}
	if meta, ok := res.(metav1.Object); ok {
		name := meta.GetName()
		if ty, ok := res.(metav1.Type); ok {
			return ty.GetKind() + " " + name
		}
		return "resource " + name
	}
	return "unnamed resource"
}
