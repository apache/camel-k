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

package trait

import (
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	sb "github.com/redhat-developer/service-binding-operator/pkg/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// The Service Binding trait allows users to connect to Provisioned Services and ServiceBindings in Kubernetes:
// https://github.com/k8s-service-bindings/spec#service-binding
// As the specification is still evolving this is subject to change
// +camel-k:trait=service-binding
type serviceBindingTrait struct {
	BaseTrait `property:",squash"`
	// List of Provisioned Services and ServiceBindings in the form KIND.VERSION.GROUP/NAME[/NAMESPACE]
	ServiceBindings []string `property:"service-bindings" json:"serviceBindings,omitempty"`
}

func newServiceBindingTrait() Trait {
	return &serviceBindingTrait{
		BaseTrait: NewBaseTrait("service-binding", 250),
	}
}

func (t *serviceBindingTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if len(t.ServiceBindings) == 0 {
		return false, nil
	}

	return e.IntegrationInPhase(
		v1.IntegrationPhaseInitialization,
		v1.IntegrationPhaseWaitingForServiceBindingCollectionReady,
		v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning,
	), nil
}

func (t *serviceBindingTrait) Apply(e *Environment) error {
	integrationServiceBindingName := e.Integration.Name + "-service-binding-request"
	services, err := t.parseProvisionedServices(e)
	if err != nil {
		return err
	}
	serviceBindings, err := t.parseServiceBindings(e)
	if err != nil {
		return err
	}
	if len(services) > 0 {
		serviceBindings = append(serviceBindings, integrationServiceBindingName)
	}
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		serviceBindingsCollectionReady := true
		for _, name := range serviceBindings {
			isIntSB := name == integrationServiceBindingName
			serviceBinding, err := t.getServiceBinding(e, name)
			// Do not throw an error if the ServiceBinding is not found and if we are managing it: we will create it
			if (err != nil && !k8serrors.IsNotFound(err)) || (err != nil && !isIntSB) {
				return err
			}
			if isIntSB {
				request := createServiceBinding(e, services, integrationServiceBindingName)
				e.Resources.Add(&request)
			}
			if isCollectionReady(serviceBinding) {
				setCollectionReady(e, name, corev1.ConditionTrue)
			} else {
				setCollectionReady(e, name, corev1.ConditionFalse)
				serviceBindingsCollectionReady = false
			}
		}
		if !serviceBindingsCollectionReady {
			e.PostProcessors = append(e.PostProcessors, func(environment *Environment) error {
				e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForServiceBindingCollectionReady
				return nil
			})
		}
		return nil
	} else if e.IntegrationInPhase(v1.IntegrationPhaseWaitingForServiceBindingCollectionReady) {
		for _, name := range serviceBindings {
			serviceBinding, err := t.getServiceBinding(e, name)
			if err != nil {
				return err
			}
			if isCollectionReady(serviceBinding) {
				setCollectionReady(e, name, corev1.ConditionTrue)
			} else {
				setCollectionReady(e, name, corev1.ConditionFalse)
				return nil
			}
		}
		e.PostActions = append(e.PostActions, func(environment *Environment) error {
			e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
			return nil
		})
	} else if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		e.ServiceBindings = make(map[string]string)
		for _, name := range serviceBindings {
			sb, err := t.getServiceBinding(e, name)
			if err != nil {
				return err
			}
			if !isCollectionReady(sb) {
				setCollectionReady(e, name, corev1.ConditionFalse)
				e.PostProcessors = append(e.PostProcessors, func(environment *Environment) error {
					e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForServiceBindingCollectionReady
					return nil
				})
				return nil
			}
			e.ServiceBindings[name] = sb.Status.Secret
			if name == integrationServiceBindingName {
				request := createServiceBinding(e, services, integrationServiceBindingName)
				e.Resources.Add(&request)
			}
		}
		e.ApplicationProperties["quarkus.kubernetes-service-binding.enabled"] = "true"
		e.ApplicationProperties["SERVICE_BINDING_ROOT"] = ServiceBindingsMountPath
	}
	return nil
}

func setCollectionReady(e *Environment, serviceBinding string, status corev1.ConditionStatus) {
	e.Integration.Status.SetCondition(
		v1.IntegrationConditionServiceBindingCollectionReady,
		status,
		"",
		fmt.Sprintf("Name=%s", serviceBinding),
	)
}

func isCollectionReady(sb sb.ServiceBinding) bool {
	for _, condition := range sb.Status.Conditions {
		if condition.Type == "CollectionReady" {
			return condition.Status == corev1.ConditionTrue && sb.Status.Secret != ""
		}
	}
	return false
}

func (t *serviceBindingTrait) getServiceBinding(e *Environment, name string) (sb.ServiceBinding, error) {
	serviceBinding := sb.ServiceBinding{}
	key := k8sclient.ObjectKey{
		Namespace: e.Integration.Namespace,
		Name:      name,
	}
	return serviceBinding, t.Client.Get(t.Ctx, key, &serviceBinding)
}

func (t *serviceBindingTrait) parseProvisionedServices(e *Environment) ([]sb.Service, error) {
	services := make([]sb.Service, 0)
	for _, s := range t.ServiceBindings {
		seg := strings.Split(s, "/")
		if !(len(seg) == 3 || len(seg) == 2) {
			return nil, fmt.Errorf("ServiceBinding: %s should be specified in the form KIND.VERSION.GROUP/NAME[/NAMESPACE]", s)
		}
		gvk := seg[0]
		index := strings.Index(gvk, ".")
		kind := seg[0][0:index]
		if kind == "ServiceBinding" {
			continue
		}
		vg := seg[0][index+1 : len(gvk)]
		index = strings.Index(vg, ".")
		version := vg[0:index]
		group := vg[index+1 : len(vg)]
		name := seg[1]
		namespace := e.Integration.Namespace
		if len(seg) == 3 {
			namespace = seg[2]
		}
		namePrefix := ""
		service := sb.Service{
			GroupVersionKind: metav1.GroupVersionKind{
				Group:   group,
				Version: version,
				Kind:    kind,
			},
			EnvVarPrefix: &namePrefix,
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Namespace: &namespace,
		}
		services = append(services, service)
	}
	return services, nil
}

func (t *serviceBindingTrait) parseServiceBindings(e *Environment) ([]string, error) {
	serviceBindings := make([]string, 0)
	for _, s := range t.ServiceBindings {
		seg := strings.Split(s, "/")
		if !(len(seg) == 3 || len(seg) == 2) {
			return nil, fmt.Errorf("ServiceBinding: %s should be specified in the form KIND.VERSION.GROUP/NAME[/NAMESPACE]", s)
		}
		gvk := seg[0]
		index := strings.Index(gvk, ".")
		kind := seg[0][0:index]
		if kind == "ServiceBinding" {
			vg := seg[0][index+1 : len(gvk)]
			if vg != "v1alpha1.operators.coreos.com" {
				return nil, fmt.Errorf("ServiceBinding: %s VERSION.GROUP should be v1alpha1.operators.coreos.com", s)
			}
			if len(seg) == 3 && seg[2] != e.Integration.Namespace {
				return nil, fmt.Errorf("ServiceBinding: %s should be in the same namespace %s as the integration", s, e.Integration.Namespace)
			}
			serviceBindings = append(serviceBindings, seg[1])
		}
	}
	return serviceBindings, nil
}

func createServiceBinding(e *Environment, services []sb.Service, name string) sb.ServiceBinding {
	spec := sb.ServiceBindingSpec{
		Services: services,
	}
	labels := map[string]string{
		v1.IntegrationLabel: e.Integration.Name,
	}
	serviceBinding := sb.ServiceBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceBinding",
			APIVersion: "operators.coreos.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: e.Integration.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: spec,
	}
	return serviceBinding
}
