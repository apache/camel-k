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

package test

import (
	"context"
	"fmt"
	"strings"

	"github.com/apache/camel-k/v2/pkg/apis"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	fakecamelclientset "github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned/fake"
	camelv1 "github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned/typed/camel/v1"
	camelv1alpha1 "github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned/typed/camel/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/util"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale"
	fakescale "k8s.io/client-go/scale/fake"
	"k8s.io/client-go/testing"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// NewFakeClient ---.
func NewFakeClient(initObjs ...runtime.Object) (client.Client, error) {
	scheme := clientscheme.Scheme

	// Setup Scheme for all resources
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c := fake.
		NewClientBuilder().
		WithScheme(scheme).
		WithIndex(
			&corev1.Pod{},
			"status.phase",
			func(obj controller.Object) []string {
				pod, _ := obj.(*corev1.Pod)
				return []string{string(pod.Status.Phase)}
			},
		).
		WithRuntimeObjects(initObjs...).
		WithStatusSubresource(&v1.IntegrationKit{}).
		Build()

	camelClientset := fakecamelclientset.NewSimpleClientset(filterObjects(scheme, initObjs, func(gvk schema.GroupVersionKind) bool {
		return strings.Contains(gvk.Group, "camel")
	})...)
	clientset := fakeclientset.NewSimpleClientset(filterObjects(scheme, initObjs, func(gvk schema.GroupVersionKind) bool {
		return !strings.Contains(gvk.Group, "camel") && !strings.Contains(gvk.Group, "knative")
	})...)
	replicasCount := make(map[string]int32)
	fakescaleclient := fakescale.FakeScaleClient{}
	fakescaleclient.AddReactor("update", "*", func(rawAction testing.Action) (bool, runtime.Object, error) {
		action := rawAction.(testing.UpdateAction)       // nolint: forcetypeassert
		obj := action.GetObject().(*autoscalingv1.Scale) // nolint: forcetypeassert
		replicas := obj.Spec.Replicas
		key := fmt.Sprintf("%s:%s:%s/%s", action.GetResource().Group, action.GetResource().Resource, action.GetNamespace(), obj.GetName())
		replicasCount[key] = replicas
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      obj.Name,
				Namespace: action.GetNamespace(),
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: replicas,
			},
		}, nil
	})
	fakescaleclient.AddReactor("get", "*", func(rawAction testing.Action) (bool, runtime.Object, error) {
		action := rawAction.(testing.GetAction) // nolint: forcetypeassert
		key := fmt.Sprintf("%s:%s:%s/%s", action.GetResource().Group, action.GetResource().Resource, action.GetNamespace(), action.GetName())
		obj := &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      action.GetName(),
				Namespace: action.GetNamespace(),
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: replicasCount[key],
			},
		}
		return true, obj, nil
	})

	return &FakeClient{
		Client:    c,
		Interface: clientset,
		camel:     camelClientset,
		scales:    &fakescaleclient,
	}, nil
}

func filterObjects(scheme *runtime.Scheme, input []runtime.Object, filter func(gvk schema.GroupVersionKind) bool) []runtime.Object {
	var res []runtime.Object
	for _, obj := range input {
		kinds, _, _ := scheme.ObjectKinds(obj)
		for _, k := range kinds {
			if filter(k) {
				res = append(res, obj)
				break
			}
		}
	}
	return res
}

// FakeClient ---.
type FakeClient struct {
	controller.Client
	kubernetes.Interface
	camel            *fakecamelclientset.Clientset
	scales           *fakescale.FakeScaleClient
	disabledGroups   []string
	enabledOpenshift bool
}

func (c *FakeClient) AddReactor(verb, resource string, reaction testing.ReactionFunc) {
	c.camel.AddReactor(verb, resource, reaction)
}

func (c *FakeClient) CamelV1() camelv1.CamelV1Interface {
	return c.camel.CamelV1()
}

func (c *FakeClient) CamelV1alpha1() camelv1alpha1.CamelV1alpha1Interface {
	return c.camel.CamelV1alpha1()
}

// GetScheme ---.
func (c *FakeClient) GetScheme() *runtime.Scheme {
	return clientscheme.Scheme
}

func (c *FakeClient) GetConfig() *rest.Config {
	return nil
}

func (c *FakeClient) GetCurrentNamespace(kubeConfig string) (string, error) {
	return "", nil
}

// Patch mimicks patch for server-side apply and simply creates the obj.
func (c *FakeClient) Patch(ctx context.Context, obj controller.Object, patch controller.Patch, opts ...controller.PatchOption) error {
	if err := c.Create(ctx, obj); err != nil {
		// Create fails if object already exists. Try to update it.
		return c.Update(ctx, obj)
	}
	return nil
}

func (c *FakeClient) DisableAPIGroupDiscovery(group string) {
	c.disabledGroups = append(c.disabledGroups, group)
}

func (c *FakeClient) EnableOpenshiftDiscovery() {
	c.enabledOpenshift = true
}

func (c *FakeClient) Discovery() discovery.DiscoveryInterface {
	return &FakeDiscovery{
		DiscoveryInterface: c.Interface.Discovery(),
		disabledGroups:     c.disabledGroups,
		enabledOpenshift:   c.enabledOpenshift,
	}
}

func (c *FakeClient) ServerOrClientSideApplier() client.ServerOrClientSideApplier {
	return client.ServerOrClientSideApplier{
		Client: c,
	}
}

func (c *FakeClient) ScalesClient() (scale.ScalesGetter, error) {
	return c.scales, nil
}

type FakeDiscovery struct {
	discovery.DiscoveryInterface
	disabledGroups   []string
	enabledOpenshift bool
}

func (f *FakeDiscovery) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	// Normalize the fake discovery to behave like the real implementation when checking for openshift
	if groupVersion == "image.openshift.io/v1" {
		if f.enabledOpenshift {
			return &metav1.APIResourceList{
				GroupVersion: "image.openshift.io/v1",
			}, nil
		} else {
			return nil, k8serrors.NewNotFound(schema.GroupResource{
				Group: "image.openshift.io",
			}, "")
		}
	}

	// used to verify if knative is installed
	if groupVersion == "serving.knative.dev/v1" && !util.StringSliceExists(f.disabledGroups, groupVersion) {
		return &metav1.APIResourceList{
			GroupVersion: "serving.knative.dev/v1",
		}, nil
	}
	if groupVersion == "eventing.knative.dev/v1" && !util.StringSliceExists(f.disabledGroups, groupVersion) {
		return &metav1.APIResourceList{
			GroupVersion: "eventing.knative.dev/v1",
		}, nil
	}
	if groupVersion == "messaging.knative.dev/v1" && !util.StringSliceExists(f.disabledGroups, groupVersion) {
		return &metav1.APIResourceList{
			GroupVersion: "messaging.knative.dev/v1",
		}, nil
	}
	if groupVersion == "messaging.knative.dev/v1beta1" && !util.StringSliceExists(f.disabledGroups, groupVersion) {
		return &metav1.APIResourceList{
			GroupVersion: "messaging.knative.dev/v1beta1",
		}, nil
	}
	return f.DiscoveryInterface.ServerResourcesForGroupVersion(groupVersion)
}
