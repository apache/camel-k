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

// Code generated by client-gen. DO NOT EDIT.

package v1beta2

import (
	"net/http"

	v1beta2 "github.com/apache/camel-k/v2/pkg/apis/duck/strimzi/v1beta2"
	"github.com/apache/camel-k/v2/pkg/client/duck/strimzi/clientset/internalclientset/scheme"
	rest "k8s.io/client-go/rest"
)

type KafkaV1beta2Interface interface {
	RESTClient() rest.Interface
	KafkasGetter
	KafkaTopicsGetter
}

// KafkaV1beta2Client is used to interact with features provided by the kafka.strimzi.io group.
type KafkaV1beta2Client struct {
	restClient rest.Interface
}

func (c *KafkaV1beta2Client) Kafkas(namespace string) KafkaInterface {
	return newKafkas(c, namespace)
}

func (c *KafkaV1beta2Client) KafkaTopics(namespace string) KafkaTopicInterface {
	return newKafkaTopics(c, namespace)
}

// NewForConfig creates a new KafkaV1beta2Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*KafkaV1beta2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new KafkaV1beta2Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*KafkaV1beta2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &KafkaV1beta2Client{client}, nil
}

// NewForConfigOrDie creates a new KafkaV1beta2Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *KafkaV1beta2Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new KafkaV1beta2Client for the given RESTClient.
func New(c rest.Interface) *KafkaV1beta2Client {
	return &KafkaV1beta2Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1beta2.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *KafkaV1beta2Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
