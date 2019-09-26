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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	serving "knative.dev/serving/pkg/apis/serving/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// A Collection is a container of Kubernetes resources
type Collection struct {
	items []runtime.Object
}

// NewCollection creates a new empty collection
func NewCollection(objcts ...runtime.Object) *Collection {
	collection := Collection{
		items: make([]runtime.Object, 0, len(objcts)),
	}

	collection.items = append(collection.items, objcts...)

	return &collection
}

// Size returns the number of resources belonging to the collection
func (c *Collection) Size() int {
	return len(c.items)
}

// Items returns all resources belonging to the collection
func (c *Collection) Items() []runtime.Object {
	return c.items
}

// AsKubernetesList returns all resources wrapped in a Kubernetes list
func (c *Collection) AsKubernetesList() *corev1.List {
	lst := corev1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: make([]runtime.RawExtension, 0, len(c.items)),
	}
	for _, res := range c.items {
		raw := runtime.RawExtension{
			Object: res,
		}
		lst.Items = append(lst.Items, raw)
	}
	return &lst
}

// Add adds a resource to the collection
func (c *Collection) Add(resource runtime.Object) {
	c.items = append(c.items, resource)
}

// AddAll adds all resources to the collection
func (c *Collection) AddAll(resource []runtime.Object) {
	c.items = append(c.items, resource...)
}

// VisitDeployment executes the visitor function on all Deployment resources
func (c *Collection) VisitDeployment(visitor func(*appsv1.Deployment)) {
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(*appsv1.Deployment); ok {
			visitor(conv)
		}
	})
}

// GetDeployment returns a Deployment that matches the given function
func (c *Collection) GetDeployment(filter func(*appsv1.Deployment) bool) *appsv1.Deployment {
	var retValue *appsv1.Deployment
	c.VisitDeployment(func(re *appsv1.Deployment) {
		if filter(re) {
			retValue = re
		}
	})
	return retValue
}

// GetDeploymentForIntegration returns a Deployment for the given integration
func (c *Collection) GetDeploymentForIntegration(integration *v1alpha1.Integration) *appsv1.Deployment {
	if integration == nil {
		return nil
	}

	return c.GetDeployment(func(d *appsv1.Deployment) bool {
		return d.ObjectMeta.Labels["camel.apache.org/integration"] == integration.Name
	})
}

// HasDeployment returns true if a deployment matching the given condition is present
func (c *Collection) HasDeployment(filter func(*appsv1.Deployment) bool) bool {
	return c.GetDeployment(filter) != nil
}

// RemoveDeployment removes and returns a Deployment that matches the given function
func (c *Collection) RemoveDeployment(filter func(*appsv1.Deployment) bool) *appsv1.Deployment {
	res := c.Remove(func(res runtime.Object) bool {
		if conv, ok := res.(*appsv1.Deployment); ok {
			return filter(conv)
		}
		return false
	})
	if res == nil {
		return nil
	}
	return res.(*appsv1.Deployment)
}

// VisitConfigMap executes the visitor function on all ConfigMap resources
func (c *Collection) VisitConfigMap(visitor func(*corev1.ConfigMap)) {
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(*corev1.ConfigMap); ok {
			visitor(conv)
		}
	})
}

// GetConfigMap returns a ConfigMap that matches the given function
func (c *Collection) GetConfigMap(filter func(*corev1.ConfigMap) bool) *corev1.ConfigMap {
	var retValue *corev1.ConfigMap
	c.VisitConfigMap(func(re *corev1.ConfigMap) {
		if filter(re) {
			retValue = re
		}
	})
	return retValue
}

// RemoveConfigMap removes and returns a ConfigMap that matches the given function
func (c *Collection) RemoveConfigMap(filter func(*corev1.ConfigMap) bool) *corev1.ConfigMap {
	res := c.Remove(func(res runtime.Object) bool {
		if conv, ok := res.(*corev1.ConfigMap); ok {
			return filter(conv)
		}
		return false
	})
	if res == nil {
		return nil
	}
	return res.(*corev1.ConfigMap)
}

// VisitService executes the visitor function on all Service resources
func (c *Collection) VisitService(visitor func(*corev1.Service)) {
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(*corev1.Service); ok {
			visitor(conv)
		}
	})
}

// GetService returns a Service that matches the given function
func (c *Collection) GetService(filter func(*corev1.Service) bool) *corev1.Service {
	var retValue *corev1.Service
	c.VisitService(func(re *corev1.Service) {
		if filter(re) {
			retValue = re
		}
	})
	return retValue
}

// GetUserServiceForIntegration returns a user Service for the given integration
func (c *Collection) GetUserServiceForIntegration(integration *v1alpha1.Integration) *corev1.Service {
	if integration == nil {
		return nil
	}
	return c.GetService(func(s *corev1.Service) bool {
		return s.ObjectMeta.Labels != nil &&
			s.ObjectMeta.Labels["camel.apache.org/integration"] == integration.Name &&
			s.ObjectMeta.Labels["camel.apache.org/service.type"] == v1alpha1.ServiceTypeUser
	})
}

// GetServiceForIntegration returns a user Service for the given integration
func (c *Collection) GetServiceForIntegration(integration *v1alpha1.Integration) *corev1.Service {
	if integration == nil {
		return nil
	}
	return c.GetService(func(s *corev1.Service) bool {
		return s.ObjectMeta.Labels != nil && s.ObjectMeta.Labels["camel.apache.org/integration"] == integration.Name
	})
}

// GetKnativeService returns a knative Service that matches the given function
func (c *Collection) GetKnativeService(filter func(*serving.Service) bool) *serving.Service {
	var retValue *serving.Service
	c.VisitKnativeService(func(re *serving.Service) {
		if filter(re) {
			retValue = re
		}
	})
	return retValue
}

// VisitRoute executes the visitor function on all Route resources
func (c *Collection) VisitRoute(visitor func(*routev1.Route)) {
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(*routev1.Route); ok {
			visitor(conv)
		}
	})
}

// GetRoute returns a Route that matches the given function
func (c *Collection) GetRoute(filter func(*routev1.Route) bool) *routev1.Route {
	var retValue *routev1.Route
	c.VisitRoute(func(re *routev1.Route) {
		if filter(re) {
			retValue = re
		}
	})
	return retValue
}

// VisitKnativeService executes the visitor function on all Knative serving Service resources
func (c *Collection) VisitKnativeService(visitor func(*serving.Service)) {
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(*serving.Service); ok {
			visitor(conv)
		}
	})
}

// GetContainer --
func (c *Collection) GetContainer(filter func(container *corev1.Container) bool) *corev1.Container {
	var retValue *corev1.Container

	c.VisitContainer(func(container *corev1.Container) {
		if filter(container) {
			retValue = container
		}
	})

	return retValue
}

// GetContainerByName --
func (c *Collection) GetContainerByName(name string) *corev1.Container {
	return c.GetContainer(func(c *corev1.Container) bool {
		return c.Name == name
	})
}

// VisitContainer executes the visitor function on all Containers inside deployments or other resources
func (c *Collection) VisitContainer(visitor func(container *corev1.Container)) {
	c.VisitDeployment(func(d *appsv1.Deployment) {
		for idx := range d.Spec.Template.Spec.Containers {
			cntref := &d.Spec.Template.Spec.Containers[idx]
			visitor(cntref)
		}
	})
	c.VisitKnativeConfigurationSpec(func(cs *servingv1.ConfigurationSpec) {
		for id := range cs.Template.Spec.Containers {
			cntref := &cs.Template.Spec.Containers[id]
			visitor(cntref)
		}
	})
}

// VisitKnativeConfigurationSpec executes the visitor function on all knative ConfigurationSpec inside serving Services
func (c *Collection) VisitKnativeConfigurationSpec(visitor func(container *servingv1.ConfigurationSpec)) {
	c.VisitKnativeService(func(s *serving.Service) {
		visitor(&s.Spec.ConfigurationSpec)
	})
}

// VisitMetaObject executes the visitor function on all meta.Object resources
func (c *Collection) VisitMetaObject(visitor func(metav1.Object)) {
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(metav1.Object); ok {
			visitor(conv)
		}
	})
}

// Visit executes the visitor function on all resources
func (c *Collection) Visit(visitor func(runtime.Object)) {
	for _, res := range c.items {
		visitor(res)
	}
}

// Remove removes the given element from the collection and returns it
func (c *Collection) Remove(selector func(runtime.Object) bool) runtime.Object {
	for idx, res := range c.items {
		if selector(res) {
			c.items = append(c.items[0:idx], c.items[idx+1:]...)
			return res
		}
	}
	return nil
}
