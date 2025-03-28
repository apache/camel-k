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
	context "context"

	strimziv1beta2 "github.com/apache/camel-k/v2/pkg/apis/duck/strimzi/v1beta2"
	scheme "github.com/apache/camel-k/v2/pkg/client/duck/strimzi/clientset/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// KafkaTopicsGetter has a method to return a KafkaTopicInterface.
// A group's client should implement this interface.
type KafkaTopicsGetter interface {
	KafkaTopics(namespace string) KafkaTopicInterface
}

// KafkaTopicInterface has methods to work with KafkaTopic resources.
type KafkaTopicInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*strimziv1beta2.KafkaTopic, error)
	List(ctx context.Context, opts v1.ListOptions) (*strimziv1beta2.KafkaTopicList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	KafkaTopicExpansion
}

// kafkaTopics implements KafkaTopicInterface
type kafkaTopics struct {
	*gentype.ClientWithList[*strimziv1beta2.KafkaTopic, *strimziv1beta2.KafkaTopicList]
}

// newKafkaTopics returns a KafkaTopics
func newKafkaTopics(c *KafkaV1beta2Client, namespace string) *kafkaTopics {
	return &kafkaTopics{
		gentype.NewClientWithList[*strimziv1beta2.KafkaTopic, *strimziv1beta2.KafkaTopicList](
			"kafkatopics",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *strimziv1beta2.KafkaTopic { return &strimziv1beta2.KafkaTopic{} },
			func() *strimziv1beta2.KafkaTopicList { return &strimziv1beta2.KafkaTopicList{} },
		),
	}
}
