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

package knative

import (
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestChannelUri(t *testing.T) {
	ref, err := ExtractObjectReference("knative:endpoint/ciao")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "",
		APIVersion: "",
		Name:       "ciao",
	}, ref)

	ref, err = ExtractObjectReference("knative:endpoint/ciao?apiVersion=xxx")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "",
		APIVersion: "xxx",
		Name:       "ciao",
	}, ref)

	ref, err = ExtractObjectReference("knative:endpoint/ciao?x=y&apiVersion=xxx")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "",
		APIVersion: "xxx",
		Name:       "ciao",
	}, ref)

	ref, err = ExtractObjectReference("knative:channel/ciao2?x=y&apiVersion=eventing.knative.dev/v1&kind=KafkaChannel")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "KafkaChannel",
		APIVersion: "eventing.knative.dev/v1",
		Name:       "ciao2",
	}, ref)

	ref, err = ExtractObjectReference("knative:endpoint/ciao?aapiVersion=xxx&kind=Broker")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Broker",
		APIVersion: "",
		Name:       "ciao",
	}, ref)

	ref, err = ExtractObjectReference("knative://endpoint/ciao?&apiVersion=serving.knative.dev/v1alpha1&kind=Service&1=1")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Service",
		APIVersion: "serving.knative.dev/v1alpha1",
		Name:       "ciao",
	}, ref)

	ref, err = ExtractObjectReference("knative://event/chuck?&apiVersion=eventing.knative.dev/v1beta1&name=broker2")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		APIVersion: "eventing.knative.dev/v1beta1",
		Name:       "broker2",
		Kind:       "Broker",
	}, ref)

	ref, err = ExtractObjectReference(
		"knative://event/chuck?&brokerApxxiVersion=eventing.knative.dev/v1beta1&brokxerName=broker2")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		Name: "default",
		Kind: "Broker",
	}, ref)

	ref, err = ExtractObjectReference("knative://event?&apiVersion=eventing.knative.dev/v1beta13&brokxerName=broker2")
	assert.Nil(t, err)
	assert.Equal(t, v1.ObjectReference{
		APIVersion: "eventing.knative.dev/v1beta13",
		Name:       "default",
		Kind:       "Broker",
	}, ref)
}

func TestNormalizeToUri(t *testing.T) {
	assert.Equal(t, "knative://channel/name.chan", NormalizeToURI(knative.CamelServiceTypeChannel, "name.chan"))
	assert.Equal(t, "knative://event/chuck", NormalizeToURI(knative.CamelServiceTypeEvent, "chuck"))
	assert.Equal(t, "knative://endpoint/xx", NormalizeToURI(knative.CamelServiceTypeEndpoint, "xx"))
	assert.Equal(t, "direct:xxx", NormalizeToURI(knative.CamelServiceTypeChannel, "direct:xxx"))
}
