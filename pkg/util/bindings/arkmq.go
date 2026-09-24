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

package bindings

import (
	"fmt"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	arkmqv1beta1 "github.com/apache/camel-k/v2/pkg/apis/duck/arkmq/v1beta1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/uri"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	RegisterBindingProvider(ArkMQBindingProvider{})
}

// camelArkMQ represents the configuration required by Camel JMS component for ArkMQ.
type camelArkMQ struct {
	queueName  string
	properties map[string]string
}

// ArkMQBindingProvider allows connecting to an ArkMQ queue via Binding.
type ArkMQBindingProvider struct{}

func (a ArkMQBindingProvider) ID() string {
	return "arkmq"
}

func (a ArkMQBindingProvider) Translate(ctx BindingContext, _ EndpointContext, endpoint camelv1.Endpoint) (*Binding, error) {
	if endpoint.Ref == nil {
		// IMPORTANT: just pass through if this provider cannot manage the binding. Another provider in the chain may take care of it.
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(endpoint.Ref.APIVersion)
	if err != nil {
		return nil, err
	}
	if gv.Group != arkmqv1beta1.ArkMQGroup {
		// IMPORTANT: just pass through if this provider cannot manage the binding. Another provider in the chain may take care of it.
		return nil, nil
	}

	camelArkMQ, err := a.toCamelArkMQ(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	arkmqURI := "jms:queue:" + camelArkMQ.queueName
	arkmqURI = uri.AppendParameters(arkmqURI, camelArkMQ.properties)

	return &Binding{
		URI: arkmqURI,
	}, nil
}

// toCamelArkMQ serializes an endpoint to a camelArkMQ struct.
func (a ArkMQBindingProvider) toCamelArkMQ(ctx BindingContext, endpoint camelv1.Endpoint) (*camelArkMQ, error) {
	switch endpoint.Ref.Kind {
	case arkmqv1beta1.ArkMQKindBroker:
		return a.fromBrokerToCamel(ctx, endpoint)
	case arkmqv1beta1.ArkMQKindAddress:
		return a.fromAddressToCamel(ctx, endpoint)
	}

	return nil, fmt.Errorf("invalid endpoint kind %q. Can only work with %s or %s kind",
		endpoint.Ref.Kind, arkmqv1beta1.ArkMQKindBroker, arkmqv1beta1.ArkMQKindAddress)
}

// Verify and transform an ActiveMQArtemis broker resource to Camel JMS queue endpoint parameters.
func (a ArkMQBindingProvider) fromBrokerToCamel(ctx BindingContext, endpoint camelv1.Endpoint) (*camelArkMQ, error) {
	props, err := endpoint.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]string)
	}

	queueName := props["destination"]
	if queueName == "" {
		queueName = props["queue"]
	}
	if queueName == "" {
		return nil, fmt.Errorf("invalid endpoint configuration: missing destination or queue property on %s %s",
			endpoint.Ref.Kind, endpoint.Ref.Name)
	}
	delete(props, "destination")
	delete(props, "queue")

	if props["brokerURL"] == "" {
		namespace := endpoint.Ref.Namespace
		if namespace == "" {
			namespace = ctx.Namespace
		}

		brokerURL, err := a.getBrokerURL(ctx, endpoint.Ref.Name, namespace)
		if err != nil {
			return nil, err
		}

		props["brokerURL"] = brokerURL
	}

	return &camelArkMQ{
		queueName:  queueName,
		properties: props,
	}, nil
}

// Verify and transform an ActiveMQArtemisAddress resource to Camel JMS queue endpoint parameters.
func (a ArkMQBindingProvider) fromAddressToCamel(ctx BindingContext, endpoint camelv1.Endpoint) (*camelArkMQ, error) {
	props, err := endpoint.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]string)
	}

	queueName := endpoint.Ref.Name
	if props["brokerURL"] == "" {
		address, err := a.lookupAddress(ctx, endpoint)
		if err != nil {
			return nil, err
		}

		if address.Spec.RoutingType == "multicast" {
			return nil, fmt.Errorf("multicast addresses (topics) are not supported on queue binding %s", endpoint.Ref.Name)
		}

		if address.Spec.QueueName != "" {
			queueName = address.Spec.QueueName
		} else if address.Spec.AddressName != "" {
			queueName = address.Spec.AddressName
		}

		brokerURL, err := a.lookupBrokerURL(ctx, address, endpoint)
		if err != nil {
			return nil, err
		}

		props["brokerURL"] = brokerURL
	}

	return &camelArkMQ{
		queueName:  queueName,
		properties: props,
	}, nil
}

func (a ArkMQBindingProvider) lookupBrokerURL(ctx BindingContext, address *arkmqv1beta1.ActiveMQArtemisAddress, endpoint camelv1.Endpoint) (string, error) {
	clusterName := address.Spec.ApplyTo
	if clusterName == "" && address.Labels != nil {
		clusterName = address.Labels[arkmqv1beta1.ArkMQBrokerLabel]
	}

	namespace := endpoint.Ref.Namespace
	if namespace == "" {
		namespace = ctx.Namespace
	}

	if clusterName == "" {
		// By default, the resource is reconciled by the existing broker in the same namespace.
		// Fallback to looking for an ActiveMQArtemis broker in the namespace.
		fallbackName, err := a.fallbackBrokerName(ctx, namespace, endpoint.Ref.Name)
		if err != nil {
			return "", err
		}
		clusterName = fallbackName
	}

	return a.getBrokerURL(ctx, clusterName, namespace)
}

func (a ArkMQBindingProvider) fallbackBrokerName(ctx BindingContext, namespace, addressName string) (string, error) {
	var brokers arkmqv1beta1.ActiveMQArtemisList
	if err := ctx.Client.List(ctx.Ctx, &brokers, ctrl.InNamespace(namespace)); err != nil {
		return "", err
	}
	if len(brokers.Items) == 0 {
		return "", fmt.Errorf("no ActiveMQArtemis broker found in namespace %s for address %s", namespace, addressName)
	}
	if len(brokers.Items) > 1 {
		return "", fmt.Errorf("multiple ActiveMQArtemis brokers found in namespace %s (%d found); specify %q label or applyTo on address %s",
			namespace, len(brokers.Items), arkmqv1beta1.ArkMQBrokerLabel, addressName)
	}

	return brokers.Items[0].Name, nil
}

func (a ArkMQBindingProvider) getBrokerURL(ctx BindingContext, clusterName, namespace string) (string, error) {
	var broker arkmqv1beta1.ActiveMQArtemis
	key := ctrl.ObjectKey{
		Namespace: namespace,
		Name:      clusterName,
	}
	if err := ctx.Client.Get(ctx.Ctx, key, &broker); err != nil {
		return "", err
	}

	port := int32(61616)
	for _, p := range broker.Status.PortStatus {
		if p.Name == "core" || p.Name == "all" || p.Name == "openwire" {
			if p.Port > 0 {
				port = p.Port
				break
			}
		}
	}

	// NOTE: this is a first implementation that follows the ArkMQ headless service naming convention (<clusterName>-hdls-svc),
	// but further development may be needed to make it more consistent.
	svcName := fmt.Sprintf("%s-hdls-svc", clusterName)
	svc, err := kubernetes.LookupService(ctx.Ctx, ctx.Client, namespace, svcName)
	if err != nil {
		return "", err
	}
	if svc == nil {
		svc, err = kubernetes.LookupService(ctx.Ctx, ctx.Client, namespace, clusterName)
		if err != nil {
			return "", err
		}
	}
	if svc == nil {
		return "", fmt.Errorf("could not find service %s in namespace %s for ArkMQ broker %s", svcName, namespace, clusterName)
	}

	for _, p := range svc.Spec.Ports {
		if p.Name == "core" || p.Name == "all" || p.Name == "openwire" || p.Port == 61616 {
			port = p.Port
			break
		}
	}

	if svc.Namespace != "" {
		return fmt.Sprintf("tcp://%s.%s.svc:%d", svc.Name, svc.Namespace, port), nil
	}
	return fmt.Sprintf("tcp://%s:%d", svc.Name, port), nil
}

func (a ArkMQBindingProvider) lookupAddress(ctx BindingContext, endpoint camelv1.Endpoint) (*arkmqv1beta1.ActiveMQArtemisAddress, error) {
	namespace := endpoint.Ref.Namespace
	if namespace == "" {
		namespace = ctx.Namespace
	}

	var address arkmqv1beta1.ActiveMQArtemisAddress
	key := ctrl.ObjectKey{
		Namespace: namespace,
		Name:      endpoint.Ref.Name,
	}
	// first check by ActiveMQArtemisAddress name
	err := ctx.Client.Get(ctx.Ctx, key, &address)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		return &address, nil
	}

	// if not found, then look at spec.queueName or spec.addressName
	var addresses arkmqv1beta1.ActiveMQArtemisAddressList
	if err := ctx.Client.List(ctx.Ctx, &addresses, ctrl.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("couldn't find any ActiveMQArtemisAddress with either name, queueName or addressName %s; error %w", endpoint.Ref.Name, err)
	}
	for i := range addresses.Items {
		item := &addresses.Items[i]
		if item.Spec.AddressName == endpoint.Ref.Name || item.Spec.QueueName == endpoint.Ref.Name {
			return item, nil
		}
	}

	return nil, fmt.Errorf("couldn't find any ActiveMQArtemisAddress with either name, queueName or addressName %s", endpoint.Ref.Name)
}

func (a ArkMQBindingProvider) Order() int {
	return OrderStandard
}
