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

// Package strimzi contains integrations with the Strimzi project for running Apache Kafka on Kubernetes
package strimzi

import (
	"fmt"

	"github.com/apache/camel-k/v2/addons/strimzi/duck/client/internalclientset"
	"github.com/apache/camel-k/v2/addons/strimzi/duck/v1beta2"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	camelv1alpha1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/util/bindings"
	"github.com/apache/camel-k/v2/pkg/util/uri"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BindingProvider allows to connect to a Kafka topic via Binding.
type BindingProvider struct {
	Client internalclientset.Interface
}

func (s BindingProvider) ID() string {
	return "strimzi"
}

func (s BindingProvider) Translate(ctx bindings.BindingContext, _ bindings.EndpointContext, endpoint camelv1.Endpoint) (*bindings.Binding, error) {
	if endpoint.Ref == nil {
		// React only on refs
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(endpoint.Ref.APIVersion)
	if err != nil {
		return nil, err
	}

	if gv.Group != v1beta2.StrimziGroup || endpoint.Ref.Kind != v1beta2.StrimziKindTopic {
		// Only operates on Strimzi Topics
		return nil, nil
	}

	props, err := endpoint.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]string)
	}

	if props["brokers"] == "" {
		// build the client if needed
		if s.Client == nil {
			kafkaClient, err := internalclientset.NewForConfig(ctx.Client.GetConfig())
			if err != nil {
				return nil, err
			}
			s.Client = kafkaClient
		}

		// look them up
		topic, err := s.Client.KafkaV1beta2().KafkaTopics(ctx.Namespace).Get(ctx.Ctx, endpoint.Ref.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		clusterName := topic.Labels[v1beta2.StrimziKafkaClusterLabel]
		if clusterName == "" {
			return nil, fmt.Errorf("no %q label defined on topic %s", v1beta2.StrimziKafkaClusterLabel, endpoint.Ref.Name)
		}

		bootstrapServers, err := s.getBootstrapServers(ctx, clusterName)
		if err != nil {
			return nil, err
		}

		props["brokers"] = bootstrapServers
	}

	kafkaURI := fmt.Sprintf("kafka:%s", endpoint.Ref.Name)
	kafkaURI = uri.AppendParameters(kafkaURI, props)

	return &bindings.Binding{
		URI: kafkaURI,
	}, nil
}

func (s BindingProvider) getBootstrapServers(ctx bindings.BindingContext, clusterName string) (string, error) {
	cluster, err := s.Client.KafkaV1beta2().Kafkas(ctx.Namespace).Get(ctx.Ctx, clusterName, v1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, l := range cluster.Status.Listeners {
		if l.Type == v1beta2.StrimziListenerTypePlain {
			if l.BootstrapServers == "" {
				return "", fmt.Errorf("cluster %q has no bootstrap servers in %q listener", clusterName, v1beta2.StrimziListenerTypePlain)
			}

			return l.BootstrapServers, nil
		}
	}

	return "", fmt.Errorf("cluster %q has no listeners of type %q", clusterName, v1beta2.StrimziListenerTypePlain)
}

// Order --.
func (s BindingProvider) Order() int {
	return bindings.OrderStandard
}

// V1alpha1BindingProvider allows to connect to a Kafka topic via Binding.
// Deprecated.
type V1alpha1BindingProvider struct {
	Client internalclientset.Interface
}

// ID --.
// Deprecated.
func (s V1alpha1BindingProvider) ID() string {
	return "strimzi"
}

// Translate --.
// Deprecated.
func (s V1alpha1BindingProvider) Translate(ctx bindings.V1alpha1BindingContext, _ bindings.V1alpha1EndpointContext, endpoint camelv1alpha1.Endpoint) (*bindings.Binding, error) {
	if endpoint.Ref == nil {
		// React only on refs
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(endpoint.Ref.APIVersion)
	if err != nil {
		return nil, err
	}

	if gv.Group != v1beta2.StrimziGroup || endpoint.Ref.Kind != v1beta2.StrimziKindTopic {
		// Only operates on Strimzi Topics
		return nil, nil
	}

	props, err := endpoint.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]string)
	}

	if props["brokers"] == "" {
		// build the client if needed
		if s.Client == nil {
			kafkaClient, err := internalclientset.NewForConfig(ctx.Client.GetConfig())
			if err != nil {
				return nil, err
			}
			s.Client = kafkaClient
		}

		// look them up
		topic, err := s.Client.KafkaV1beta2().KafkaTopics(ctx.Namespace).Get(ctx.Ctx, endpoint.Ref.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		clusterName := topic.Labels[v1beta2.StrimziKafkaClusterLabel]
		if clusterName == "" {
			return nil, fmt.Errorf("no %q label defined on topic %s", v1beta2.StrimziKafkaClusterLabel, endpoint.Ref.Name)
		}

		bootstrapServers, err := s.getBootstrapServers(ctx, clusterName)
		if err != nil {
			return nil, err
		}

		props["brokers"] = bootstrapServers
	}

	kafkaURI := fmt.Sprintf("kafka:%s", endpoint.Ref.Name)
	kafkaURI = uri.AppendParameters(kafkaURI, props)

	return &bindings.Binding{
		URI: kafkaURI,
	}, nil
}

// getBootstrapServers --.
// Deprecated.
func (s V1alpha1BindingProvider) getBootstrapServers(ctx bindings.V1alpha1BindingContext, clusterName string) (string, error) {
	cluster, err := s.Client.KafkaV1beta2().Kafkas(ctx.Namespace).Get(ctx.Ctx, clusterName, v1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, l := range cluster.Status.Listeners {
		if l.Type == v1beta2.StrimziListenerTypePlain {
			if l.BootstrapServers == "" {
				return "", fmt.Errorf("cluster %q has no bootstrap servers in %q listener", clusterName, v1beta2.StrimziListenerTypePlain)
			}

			return l.BootstrapServers, nil
		}
	}

	return "", fmt.Errorf("cluster %q has no listeners of type %q", clusterName, v1beta2.StrimziListenerTypePlain)
}

// Order --.
// Deprecated.
func (s V1alpha1BindingProvider) Order() int {
	return bindings.OrderStandard
}
