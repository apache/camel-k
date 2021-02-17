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

	"github.com/apache/camel-k/addons/strimzi/duck/v1beta1"
	"github.com/apache/camel-k/addons/strimzi/duck/v1beta1/client/internalclientset"
	typedclient "github.com/apache/camel-k/addons/strimzi/duck/v1beta1/client/internalclientset/typed/duck/v1beta1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/bindings"
	"github.com/apache/camel-k/pkg/util/uri"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// StrimziBindingProvider allows to connect to a Kafka topic via KameletBinding
type StrimziBindingProvider struct {
	Client typedclient.KafkaV1beta1Interface
}

func (s StrimziBindingProvider) ID() string {
	return "strimzi"
}

func (s StrimziBindingProvider) Translate(ctx bindings.BindingContext, _ bindings.EndpointContext, endpoint v1alpha1.Endpoint) (*bindings.Binding, error) {
	if endpoint.Ref == nil {
		// React only on refs
		return nil, nil
	}
	gv, err := schema.ParseGroupVersion(endpoint.Ref.APIVersion)
	if err != nil {
		return nil, err
	}

	if gv.Group != v1beta1.StrimziGroup || endpoint.Ref.Kind != v1beta1.StrimziKindTopic {
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
			s.Client = kafkaClient.KafkaV1beta1()
		}

		// look them up
		topic, err := s.Client.KafkaTopics(ctx.Namespace).Get(ctx.Ctx, endpoint.Ref.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		clusterName := topic.Labels[v1beta1.StrimziKafkaClusterLabel]
		if clusterName == "" {
			return nil, fmt.Errorf("no %q label defined on topic %s", v1beta1.StrimziKafkaClusterLabel, endpoint.Ref.Name)
		}

		cluster, err := s.Client.Kafkas(ctx.Namespace).Get(ctx.Ctx, clusterName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		var listener *v1beta1.KafkaStatusListener
		for _, l := range cluster.Status.Listeners {
			if l.Type == v1beta1.StrimziListenerTypePlain {
				listener = &l
				break
			}
		}

		if listener == nil {
			return nil, fmt.Errorf("cluster %q has no listeners of type %q", clusterName, v1beta1.StrimziListenerTypePlain)
		}
		if listener.BootstrapServers == "" {
			return nil, fmt.Errorf("cluster %q has no bootstrap servers in %q listener", clusterName, v1beta1.StrimziListenerTypePlain)
		}
		props["brokers"] = listener.BootstrapServers
	}

	kafkaURI := fmt.Sprintf("kafka:%s", endpoint.Ref.Name)
	kafkaURI = uri.AppendParameters(kafkaURI, props)

	return &bindings.Binding{
		URI: kafkaURI,
	}, nil
}

func (s StrimziBindingProvider) Order() int {
	return bindings.OrderStandard
}
