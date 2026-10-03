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
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/metadata"
)

const (
	serviceKind       = "Service"
	serviceTraitID    = "service"
	serviceTraitOrder = 1500
)

type serviceTrait struct {
	BaseTrait
	traitv1.ServiceTrait `property:",squash"`

	servicePorts []servicePort
}

// servicePort is supporting port parsing.
type servicePort struct {
	name          string
	containerPort int32
	port          int32
	protocol      string
}

func newServiceTrait() Trait {
	return &serviceTrait{
		BaseTrait: NewBaseTrait(serviceTraitID, serviceTraitOrder),
	}
}

func (t *serviceTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationCondition(
			serviceKind,
			v1.IntegrationConditionServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionServiceNotAvailableReason,
			"explicitly disabled",
		), nil
	}
	// in case the knative-service and service trait are enabled, the knative-service has priority
	// then this service is disabled
	if e.GetTrait(knativeServiceTraitID) != nil {
		knativeServiceTrait, _ := e.GetTrait(knativeServiceTraitID).(*knativeServiceTrait)
		if ptr.Deref(knativeServiceTrait.Enabled, true) {
			return false, NewIntegrationConditionPlatformDisabledWithMessage(serviceKind, "knative-service trait has priority over this trait"), nil
		}
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	enabled := ptr.Deref(t.Enabled, false)
	if ptr.Deref(t.Auto, true) {
		exposeHTTPServices, err := e.ConsumeMeta(false, func(meta metadata.IntegrationMetadata) bool {
			return meta.ExposesHTTPServices
		})
		var condition *TraitCondition
		if err != nil {
			condition = NewIntegrationCondition(
				serviceKind,
				v1.IntegrationConditionServiceAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionServiceNotAvailableReason,
				err.Error(),
			)

			return false, condition, err
		}

		enabled = exposeHTTPServices
	}

	servicePorts, err := t.parseServicePorts()
	if err != nil {
		return false, nil, err
	}
	t.servicePorts = servicePorts

	return enabled || len(t.servicePorts) > 0, nil, nil
}

func (t *serviceTrait) Apply(e *Environment) error {
	svc := e.Resources.GetServiceForIntegration(e.Integration)
	// add a new service if not already created
	if svc == nil {
		svc = t.getServiceFor(e.Integration.Name, e.Integration.Namespace)

		var serviceType corev1.ServiceType
		if t.Type != nil {
			switch *t.Type {
			case traitv1.ServiceTypeClusterIP:
				serviceType = corev1.ServiceTypeClusterIP
			case traitv1.ServiceTypeNodePort:
				serviceType = corev1.ServiceTypeNodePort
			case traitv1.ServiceTypeLoadBalancer:
				serviceType = corev1.ServiceTypeLoadBalancer
			default:
				return fmt.Errorf("unsupported service type: %s", *t.Type)
			}
		} else if ptr.Deref(t.NodePort, false) { //nolint:staticcheck
			t.L.ForIntegration(e.Integration).Infof("Integration %s/%s should no more use the flag node-port as it is deprecated, use type instead", e.Integration.Namespace, e.Integration.Name)
			serviceType = corev1.ServiceTypeNodePort
		}
		svc.Spec.Type = serviceType
	}
	var networkPolicy *networkingv1.NetworkPolicy
	if ptr.Deref(t.NetworkPolicyEnabled, false) {
		var err error
		networkPolicy, err = t.getNetworkPolicyFor(e.Integration.Name, e.Integration.Namespace)
		if err != nil {
			return err
		}
	}

	e.Resources.Add(svc)
	if networkPolicy != nil {
		e.Resources.Add(networkPolicy)
	}

	return nil
}

func (t *serviceTrait) getNetworkPolicyFor(itName, itNamespace string) (*networkingv1.NetworkPolicy, error) {
	if len(t.NetworkPolicyNamespaceSelector) == 0 && len(t.NetworkPolicyPodSelector) == 0 {
		return nil, errors.New("network policy is enabled but no namespace or pod selector is configured")
	}

	from := networkingv1.NetworkPolicyPeer{}
	if len(t.NetworkPolicyNamespaceSelector) > 0 {
		from.NamespaceSelector = &metav1.LabelSelector{
			MatchLabels: maps.Clone(t.NetworkPolicyNamespaceSelector),
		}
	}
	if len(t.NetworkPolicyPodSelector) > 0 {
		from.PodSelector = &metav1.LabelSelector{
			MatchLabels: maps.Clone(t.NetworkPolicyPodSelector),
		}
	}

	policy := &networkingv1.NetworkPolicy{
		Name:      itName,
		Namespace: itNamespace,
		Labels: map[string]string{
			v1.IntegrationLabel: itName,
		},
		Kind:       "NetworkPolicy",
		APIVersion: networkingv1.SchemeGroupVersion.String(),
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					v1.IntegrationLabel: itName,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{From: []networkingv1.NetworkPolicyPeer{from}},
			},
		},
	}

	return policy, nil
}

func (t *serviceTrait) getServiceFor(itName, itNamespace string) *corev1.Service {
	labels := map[string]string{
		v1.IntegrationLabel: itName,
	}
	maps.Copy(labels, t.Labels)
	ports := t.getServicePorts()

	return &corev1.Service{
		Kind:        "Service",
		APIVersion:  "v1",
		Name:        itName,
		Namespace:   itNamespace,
		Labels:      labels,
		Annotations: t.Annotations,
		Spec: corev1.ServiceSpec{
			Ports: ports,
			Selector: map[string]string{
				v1.IntegrationLabel: itName,
			},
		},
	}
}

func (t *serviceTrait) parseServicePorts() ([]servicePort, error) {
	servicePorts := make([]servicePort, 0, len(t.Ports))
	for _, port := range t.Ports {
		portSplit := strings.Split(port, ";")
		if len(portSplit) < 3 {
			return nil, fmt.Errorf("could not parse service port %s properly: expected format "+
				"\"port-name;port-number;container-port-number[;port-protocol]\"", port)
		}
		servicePortInt32, err := strconv.ParseInt(portSplit[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("could not parse port number in %s properly: expected port-number as a number", port)
		}
		containerPortInt32, err := strconv.ParseInt(portSplit[2], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("could not parse container port number in %s properly: expected container-port-number as a number", port)
		}
		sp := servicePort{
			name:          portSplit[0],
			containerPort: int32(containerPortInt32),
			port:          int32(servicePortInt32),
		}
		protocol := "TCP"
		if len(portSplit) > 3 {
			protocol = portSplit[3]
		}
		sp.protocol = protocol
		servicePorts = append(servicePorts, sp)
	}

	return servicePorts, nil
}

func (t *serviceTrait) getServicePorts() []corev1.ServicePort {
	ports := make([]corev1.ServicePort, 0, len(t.servicePorts))
	for _, port := range t.servicePorts {
		p := corev1.ServicePort{
			Name: port.name,
			Port: port.port,
			TargetPort: intstr.IntOrString{
				IntVal: port.containerPort,
			},
			Protocol: corev1.Protocol(port.protocol),
		}
		ports = append(ports, p)
	}

	return ports
}
