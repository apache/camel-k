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
	"errors"
	"fmt"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	strimziv1 "github.com/apache/camel-k/v2/pkg/apis/duck/strimzi/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/uri"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	RegisterBindingProvider(StrimziBindingProvider{})
}

// camelKafka represent the configuration required by Camel Kafka component.
type camelKafka struct {
	topicName  string
	properties map[string]string
}

// StrimziBindingProvider allows to connect to a Kafka topic via Binding.
type StrimziBindingProvider struct{}

func (s StrimziBindingProvider) ID() string {
	return "strimzi"
}

func (s StrimziBindingProvider) Translate(ctx BindingContext, _ EndpointContext, endpoint camelv1.Endpoint) (*Binding, error) {
	if endpoint.Ref == nil {
		// IMPORTANT: just pass through if this provider cannot manage the binding. Another provider in the chain may take care or it.
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(endpoint.Ref.APIVersion)
	if err != nil {
		return nil, err
	}
	if gv.Group != strimziv1.StrimziGroup {
		// IMPORTANT: just pass through if this provider cannot manage the binding. Another provider in the chain may take care or it.
		return nil, nil
	}

	camelKafka, err := s.toCamelKafka(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	kafkaURI := "kafka:" + camelKafka.topicName
	kafkaURI = uri.AppendParameters(kafkaURI, camelKafka.properties)

	return &Binding{
		URI: kafkaURI,
	}, nil
}

// toCamelKafka serialize an endpoint to a camelKafka struct.
func (s StrimziBindingProvider) toCamelKafka(ctx BindingContext, endpoint camelv1.Endpoint) (*camelKafka, error) {
	switch endpoint.Ref.Kind {
	case strimziv1.StrimziKindKafkaCluster:
		return s.fromKafkaToCamel(ctx, endpoint)
	case strimziv1.StrimziKindTopic:
		return s.fromKafkaTopicToCamel(ctx, endpoint)
	}

	return nil, fmt.Errorf("invalid endpoint kind. Can only work with %s or %s kind", strimziv1.StrimziKindKafkaCluster, strimziv1.StrimziKindTopic)
}

// Verify and transform a Kafka resource to Camel Kafka endpoint parameters.
func (s StrimziBindingProvider) fromKafkaToCamel(ctx BindingContext, endpoint camelv1.Endpoint) (*camelKafka, error) {
	props, err := endpoint.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil || props["topic"] == "" {
		return nil, errors.New("invalid endpoint configuration: missing topic property")
	}
	topicName := props["topic"]
	delete(props, "topic")
	if props["brokers"] == "" {
		namespace := endpoint.Ref.Namespace
		if namespace == "" {
			namespace = ctx.Namespace
		}

		bootstrapServers, err := s.getBootstrapServers(ctx, endpoint.Ref.Name, namespace)
		if err != nil {
			return nil, err
		}
		props["brokers"] = bootstrapServers
	}

	return &camelKafka{
		topicName:  topicName,
		properties: props,
	}, nil
}

// Verify and transform a KafkaTopic resource to Camel Kafka endpoint parameters.
func (s StrimziBindingProvider) fromKafkaTopicToCamel(ctx BindingContext, endpoint camelv1.Endpoint) (*camelKafka, error) {
	props, err := endpoint.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]string)
	}
	if props["brokers"] == "" {
		bootstrapServers, err := s.lookupBootstrapServers(ctx, endpoint)
		if err != nil {
			return nil, err
		}

		props["brokers"] = bootstrapServers
	}

	return &camelKafka{
		topicName:  endpoint.Ref.Name,
		properties: props,
	}, nil
}

func (s StrimziBindingProvider) lookupBootstrapServers(ctx BindingContext, endpoint camelv1.Endpoint) (string, error) {
	topic, err := s.lookupTopic(ctx, endpoint)
	if err != nil {
		return "", err
	}

	clusterName := topic.Labels[strimziv1.StrimziKafkaClusterLabel]
	if clusterName == "" {
		return "", fmt.Errorf("no %q label defined on topic %s", strimziv1.StrimziKafkaClusterLabel, endpoint.Ref.Name)
	}
	namespace := endpoint.Ref.Namespace
	if namespace == "" {
		namespace = ctx.Namespace
	}
	bootstrapServers, err := s.getBootstrapServers(ctx, clusterName, namespace)
	if err != nil {
		return "", err
	}

	return bootstrapServers, nil
}

func (s StrimziBindingProvider) getBootstrapServers(ctx BindingContext, clusterName, namespace string) (string, error) {
	// Read without the manager cache to preserve live, cross-namespace lookups without requiring watch permissions.
	object, err := kubernetes.GetUnstructured(ctx.Ctx, ctx.Client,
		strimziv1.SchemeGroupVersion.WithKind(strimziv1.StrimziKindKafkaCluster), clusterName, namespace)
	if err != nil {
		return "", err
	}
	cluster := strimziv1.Kafka{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &cluster); err != nil {
		return "", err
	}

	for _, l := range cluster.Status.Listeners {
		if l.Name == strimziv1.StrimziListenerNamePlain {
			if l.BootstrapServers == "" {
				return "", fmt.Errorf("cluster %q has no bootstrap servers in %q listener", clusterName, strimziv1.StrimziListenerNamePlain)
			}

			return l.BootstrapServers, nil
		}
	}

	return "", fmt.Errorf("cluster %q has no listeners of name %q", clusterName, strimziv1.StrimziListenerNamePlain)
}

func (s StrimziBindingProvider) lookupTopic(ctx BindingContext, endpoint camelv1.Endpoint) (*strimziv1.KafkaTopic, error) {
	namespace := endpoint.Ref.Namespace
	if namespace == "" {
		namespace = ctx.Namespace
	}
	// first check by KafkaTopic name
	object, err := kubernetes.GetUnstructured(ctx.Ctx, ctx.Client,
		strimziv1.SchemeGroupVersion.WithKind(strimziv1.StrimziKindTopic), endpoint.Ref.Name, namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		topic := strimziv1.KafkaTopic{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &topic); err != nil {
			return nil, err
		}

		return &topic, nil
	}

	// if not found, then, look at the .status.topicName (it may be autogenerated)
	// Unstructured lists also bypass the cache, without introducing a custom field index.
	objects := unstructured.UnstructuredList{}
	objects.SetGroupVersionKind(strimziv1.SchemeGroupVersion.WithKind("KafkaTopicList"))
	err = ctx.Client.List(ctx.Ctx, &objects, client.InNamespace(namespace))

	if err != nil {
		return nil, fmt.Errorf("couldn't find any KafkaTopic with either name or topicName %s; error %w", endpoint.Ref.Name, err)
	}
	for i := range objects.Items {
		topic := strimziv1.KafkaTopic{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objects.Items[i].Object, &topic); err != nil {
			return nil, err
		}
		if topic.Status.TopicName == endpoint.Ref.Name {
			return &topic, nil
		}
	}

	return nil, fmt.Errorf("couldn't find any KafkaTopic with either name or topicName %s", endpoint.Ref.Name)
}

// Order --.
func (s StrimziBindingProvider) Order() int {
	return OrderStandard
}
