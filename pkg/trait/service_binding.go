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

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	sb "github.com/redhat-developer/service-binding-operator/api/v1alpha1"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/builder"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/context"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/handler/collect"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/handler/mapping"
	"github.com/redhat-developer/service-binding-operator/pkg/reconcile/pipeline/handler/naming"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/reference"
)

// The Service Binding trait allows users to connect to Services in Kubernetes:
// https://github.com/k8s-service-bindings/spec#service-binding
// As the specification is still evolving this is subject to change
// +camel-k:trait=service-binding
type serviceBindingTrait struct {
	BaseTrait `property:",squash"`
	// List of Services in the form [[apigroup/]version:]kind:[namespace/]name
	Services []string `property:"services" json:"services,omitempty"`
}

func newServiceBindingTrait() Trait {
	return &serviceBindingTrait{
		BaseTrait: NewBaseTrait("service-binding", 250),
	}
}

func (t *serviceBindingTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	if len(t.Services) == 0 {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseWaitingForBindings) ||
		e.IntegrationInRunningPhases(), nil
}

func (t *serviceBindingTrait) Apply(e *Environment) error {
	services, err := t.parseServices(e)
	if err != nil {
		return err
	}

	serviceBindingCrd := []sb.ServiceBinding{}

	for _, name := range services {
		serviceBinding := createServiceBinding(e, services, e.Integration.Name)
		append(serviceBindingCrd, serviceBinding)
	}

	var camelKFlow = []pipeline.Handler{
		pipeline.HandlerFunc(collect.PreFlight),
		pipeline.HandlerFunc(collect.ProvisionedService),
		pipeline.HandlerFunc(collect.BindingDefinitions),
		pipeline.HandlerFunc(collect.BindingItems),
		pipeline.HandlerFunc(collect.OwnedResources),
		pipeline.HandlerFunc(mapping.Handle),
		pipeline.HandlerFunc(naming.Handle),
	}

	p := builder.Builder().WithHandlers(camelKFlow...).WithContextProvider(context.Provider(e.Client, context.ResourceLookup(e.Client.RESTMapper()))).Build()

	p.Process(serviceBindingCrd)
	// construct Secret
	name, secretExist := i.bindingSecretName()
	data := i.bindingItemMap()
	if len(data) == 0 {
		return "", nil
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.bindingMeta.Namespace,
			Name:      name,
		},
		StringData: data,
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		serviceBindingsCollectionReady := true
		for _, name := range serviceBindings {
			isIntSB := name == e.Integration.Name
			serviceBinding, err := t.getServiceBinding(e, name)
			// Do not throw an error if the ServiceBinding is not found and if we are managing it: we will create it
			if (err != nil && !k8serrors.IsNotFound(err)) || (err != nil && !isIntSB) {
				return err
			}
			if isIntSB {
				request := createServiceBinding(e, services, e.Integration.Name)
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
				e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForBindings
				return nil
			})
		}
		return nil
	} else if e.IntegrationInPhase(v1.IntegrationPhaseWaitingForBindings) {
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
			if name == e.Integration.Name {
				request := createServiceBinding(e, services, name)
				e.Resources.Add(&request)
			}
		}
	} else if e.IntegrationInRunningPhases() {
		e.ServiceBindings = make(map[string]string)
		for _, name := range serviceBindings {
			sb, err := t.getServiceBinding(e, name)
			if err != nil {
				return err
			}
			if !isCollectionReady(sb) {
				setCollectionReady(e, name, corev1.ConditionFalse)
				e.PostProcessors = append(e.PostProcessors, func(environment *Environment) error {
					e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForBindings
					return nil
				})
				return nil
			}
			e.ServiceBindings[name] = sb.Status.Secret
			if name == e.Integration.Name {
				request := createServiceBinding(e, services, name)
				e.Resources.Add(&request)
			}
		}
		e.ApplicationProperties["quarkus.kubernetes-service-binding.enabled"] = "true"
		e.ApplicationProperties["SERVICE_BINDING_ROOT"] = serviceBindingsMountPath
	}
	return nil
}

func (i *impl) Process(binding interface{}) (bool, error) {
	ctx, err := i.ctxProvider.Get(binding)
	if err != nil {
		return false, err
	}
	var status pipeline.FlowStatus
	for _, h := range i.handlers {
		h.Handle(ctx)
		status = ctx.FlowStatus()
		if status.Stop {
			break
		}
	}

	return status.Retry, status.Err
}

func setCollectionReady(e *Environment, serviceBinding string, status corev1.ConditionStatus) {
	e.Integration.Status.SetCondition(
		v1.IntegrationConditionServiceBindingsCollectionReady,
		status,
		"",
		fmt.Sprintf("Name=%s", serviceBinding),
	)
}

func isCollectionReady(sb sb.ServiceBinding) bool {
	for _, condition := range sb.Status.Conditions {
		if condition.Type == "CollectionReady" {
			return condition.Status == metav1.ConditionTrue && sb.Status.Secret != ""
		}
	}
	return false
}

func (t *serviceBindingTrait) getServiceBinding(e *Environment, name string) (sb.ServiceBinding, error) {
	serviceBinding := sb.ServiceBinding{}
	key := ctrl.ObjectKey{
		Namespace: e.Integration.Namespace,
		Name:      name,
	}
	return serviceBinding, t.Client.Get(e.Ctx, key, &serviceBinding)
}

func (t *serviceBindingTrait) parseServices(e *Environment) ([]sb.Service, error) {
	services := make([]sb.Service, 0)
	converter := reference.NewConverter("")
	for _, s := range t.Services {
		ref, err := converter.FromString(s)
		if err != nil {
			return services, err
		}
		namespace := e.Integration.Namespace
		if ref.Namespace != "" {
			namespace = ref.Namespace
		}
		service := sb.Service{
			NamespacedRef: sb.NamespacedRef{
				Ref: sb.Ref{
					Group:   ref.GroupVersionKind().Group,
					Version: ref.GroupVersionKind().Version,
					Kind:    ref.Kind,
					Name:    ref.Name,
				},
				Namespace: &namespace,
			},
		}
		services = append(services, service)
	}
	return services, nil
}

func createServiceBinding(e *Environment, services []sb.Service, name string) sb.ServiceBinding {
	spec := sb.ServiceBindingSpec{
		NamingStrategy: "none",
		Services:       services,
	}
	labels := map[string]string{
		v1.IntegrationLabel: e.Integration.Name,
	}
	serviceBinding := sb.ServiceBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceBinding",
			APIVersion: "binding.operators.coreos.com/v1alpha1",
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