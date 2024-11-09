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
	"github.com/apache/camel-k/v2/pkg/apis/duck/strimzi/v1beta2"
	"github.com/apache/camel-k/v2/pkg/client/duck/strimzi/clientset/internalclientset"
	"github.com/apache/camel-k/v2/pkg/util/uri"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	RegisterBindingProvider(StrimziBindingProvider{})
}

// camelKafka represent the configuration required by Camel Kafka component.
type camelKafka struct {
	topicName  string
	properties map[string]string
}

// BindingProvider allows to connect to a Kafka topic via Binding.
type StrimziBindingProvider struct {
	Client internalclientset.Interface
}

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
	if gv.Group != v1beta2.StrimziGroup {
		// IMPORTANT: just pass through if this provider cannot manage the binding. Another provider in the chain may take care or it.
		return nil, nil
	}

	camelKafka, err := s.toCamelKafka(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	kafkaURI := fmt.Sprintf("kafka:%s", camelKafka.topicName)
	kafkaURI = uri.AppendParameters(kafkaURI, camelKafka.properties)

	return &Binding{
		URI: kafkaURI,
	}, nil
}

// toCamelKafka serialize an endpoint to a camelKafka struct.
func (s StrimziBindingProvider) toCamelKafka(ctx BindingContext, endpoint camelv1.Endpoint) (*camelKafka, error) {
	switch endpoint.Ref.Kind {
	case v1beta2.StrimziKindKafkaCluster:
		return s.fromKafkaToCamel(ctx, endpoint)
	case v1beta2.StrimziKindTopic:
		return s.fromKafkaTopicToCamel(ctx, endpoint)
	}

	return nil, fmt.Errorf("invalid endpoint kind. Can only work with %s or %s kind", v1beta2.StrimziKindKafkaCluster, v1beta2.StrimziKindTopic)
}

// Verify and transform a Kafka resource to Camel Kafka endpoint parameters.
func (s StrimziBindingProvider) fromKafkaToCamel(ctx BindingContext, endpoint camelv1.Endpoint) (*camelKafka, error) {
	props, err := endpoint.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil || props["topic"] == "" {
		return nil, fmt.Errorf("invalid endpoint configuration: missing topic property")
	}
	topicName := props["topic"]
	delete(props, "topic")
	if props["brokers"] == "" {
		// build the client if needed
		if s.Client == nil {
			kafkaClient, err := internalclientset.NewForConfig(ctx.Client.GetConfig())
			if err != nil {
				return nil, err
			}
			s.Client = kafkaClient
		}
		bootstrapServers, err := s.getBootstrapServers(ctx, endpoint.Ref.Name)
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
	// build the client if needed
	if s.Client == nil {
		kafkaClient, err := internalclientset.NewForConfig(ctx.Client.GetConfig())
		if err != nil {
			return "", err
		}
		s.Client = kafkaClient
	}

	topic, err := s.lookupTopic(ctx, endpoint)
	if err != nil {
		return "", err
	}

	clusterName := topic.Labels[v1beta2.StrimziKafkaClusterLabel]
	if clusterName == "" {
		return "", fmt.Errorf("no %q label defined on topic %s", v1beta2.StrimziKafkaClusterLabel, endpoint.Ref.Name)
	}
	bootstrapServers, err := s.getBootstrapServers(ctx, clusterName)
	if err != nil {
		return "", err
	}

	return bootstrapServers, nil
}

func (s StrimziBindingProvider) getBootstrapServers(ctx BindingContext, clusterName string) (string, error) {
	cluster, err := s.Client.KafkaV1beta2().Kafkas(ctx.Namespace).Get(ctx.Ctx, clusterName, v1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, l := range cluster.Status.Listeners {
		if l.Name == v1beta2.StrimziListenerNamePlain {
			if l.BootstrapServers == "" {
				return "", fmt.Errorf("cluster %q has no bootstrap servers in %q listener", clusterName, v1beta2.StrimziListenerNamePlain)
			}

			return l.BootstrapServers, nil
		}
	}

	return "", fmt.Errorf("cluster %q has no listeners of name %q", clusterName, v1beta2.StrimziListenerNamePlain)
}

func (s StrimziBindingProvider) lookupTopic(ctx BindingContext, endpoint camelv1.Endpoint) (*v1beta2.KafkaTopic, error) {
	// first check by KafkaTopic name
	topic, err := s.Client.KafkaV1beta2().KafkaTopics(ctx.Namespace).Get(ctx.Ctx, endpoint.Ref.Name, v1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		return topic, nil
	}

	// if not found, then, look at the .status.topicName (it may be autogenerated)
	topics, err := s.Client.KafkaV1beta2().KafkaTopics(ctx.Namespace).List(ctx.Ctx, v1.ListOptions{
		FieldSelector: "status.topicName=" + endpoint.Ref.Name,
	})

	if err != nil {
		return nil, err
	}
	if len(topics.Items) == 0 {
		return nil, fmt.Errorf("couldn't find any KafkaTopic with either name or topicName %s", endpoint.Ref.Name)
	}

	return &topics.Items[0], nil
}

// Order --.
func (s StrimziBindingProvider) Order() int {
	return OrderStandard
}
