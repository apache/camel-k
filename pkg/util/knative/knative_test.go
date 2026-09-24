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
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
	eventing "github.com/apache/camel-k/v2/pkg/apis/duck/knative/eventing/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
)

func TestCreateSinkBinding(t *testing.T) {
	sb := CreateSinkBinding(
		corev1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "my-source", Namespace: "my-ns"},
		corev1.ObjectReference{APIVersion: "messaging.knative.dev/v1", Kind: "Channel", Name: "my-channel"},
	)

	assert.Equal(t, "my-ns", sb.Namespace)
	assert.Equal(t, "my-source", sb.Name)
	assert.Equal(t, "Deployment", sb.Spec.Subject.Kind)
	assert.Equal(t, "my-source", sb.Spec.Subject.Name)
	require.NotNil(t, sb.Spec.Sink.Ref)
	assert.Equal(t, "Channel", sb.Spec.Sink.Ref.Kind)
	assert.Equal(t, "my-channel", sb.Spec.Sink.Ref.Name)

	actual, err := json.Marshal(sb)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"kind": "SinkBinding",
		"apiVersion": "sources.knative.dev/v1",
		"metadata": {"name": "my-source", "namespace": "my-ns"},
		"spec": {
			"sink": {"ref": {"kind": "Channel", "name": "my-channel", "apiVersion": "messaging.knative.dev/v1"}},
			"subject": {"apiVersion": "apps/v1", "kind": "Deployment", "name": "my-source"}
		},
		"status": {}
	}`, string(actual))
}

func TestCreateSubscription(t *testing.T) {
	sub := CreateSubscription(
		corev1.ObjectReference{APIVersion: "messaging.knative.dev/v1", Kind: "Channel", Name: "my-channel", Namespace: "my-ns"},
		"my-service", "/channels/my-channel",
	)

	assert.Equal(t, "my-ns", sub.Namespace)
	assert.Equal(t, "my-channel-my-service", sub.Name)
	assert.Equal(t, "my-channel", sub.Spec.Channel.Name)
	require.NotNil(t, sub.Spec.Subscriber)
	require.NotNil(t, sub.Spec.Subscriber.Ref)
	assert.Equal(t, "my-service", sub.Spec.Subscriber.Ref.Name)
	assert.Equal(t, "/channels/my-channel", sub.Spec.Subscriber.URI.String())

	actual, err := json.Marshal(sub)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"kind": "Subscription",
		"apiVersion": "messaging.knative.dev/v1",
		"metadata": {"name": "my-channel-my-service", "namespace": "my-ns"},
		"spec": {
			"channel": {"kind": "Channel", "name": "my-channel", "apiVersion": "messaging.knative.dev/v1"},
			"subscriber": {
				"ref": {"kind": "Service", "name": "my-service", "apiVersion": "serving.knative.dev/v1"},
				"uri": "/channels/my-channel"
			}
		},
		"status": {"physicalSubscription": {}}
	}`, string(actual))
}

func TestCreateKnativeServiceTrigger(t *testing.T) {
	trigger, err := CreateKnativeServiceTrigger(
		corev1.ObjectReference{APIVersion: "eventing.knative.dev/v1", Kind: "Broker", Name: "default", Namespace: "my-ns"},
		"my-service", "evt.type", "/events/evt.type", map[string]string{"type": "evt.type"},
	)
	require.NoError(t, err)

	assert.Equal(t, "my-ns", trigger.Namespace)
	assert.Equal(t, "default-my-service-evttype", trigger.Name)
	assert.Equal(t, "default", trigger.Spec.Broker)
	require.NotNil(t, trigger.Spec.Subscriber.Ref)
	assert.Equal(t, "my-service", trigger.Spec.Subscriber.Ref.Name)
	assert.Equal(t, "/events/evt.type", trigger.Spec.Subscriber.URI.String())
	require.NotNil(t, trigger.Spec.Filter)
	assert.Equal(t, "evt.type", trigger.Spec.Filter.Attributes["type"])

	actual, err := json.Marshal(trigger)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"kind": "Trigger",
		"apiVersion": "eventing.knative.dev/v1",
		"metadata": {"name": "default-my-service-evttype", "namespace": "my-ns"},
		"spec": {
			"broker": "default",
			"filter": {"attributes": {"type": "evt.type"}},
			"subscriber": {
				"ref": {"kind": "Service", "name": "my-service", "apiVersion": "serving.knative.dev/v1"},
				"uri": "/events/evt.type"
			}
		},
		"status": {}
	}`, string(actual))
}

func TestGetSinkURL(t *testing.T) {
	brokerURL, err := apis.ParseURL("http://broker-ingress.knative-eventing.svc.cluster.local/my-ns/default")
	require.NoError(t, err)
	client, err := internal.NewFakeClient(
		&eventing.Broker{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Broker",
				APIVersion: eventing.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "default",
			},
			Status: eventing.BrokerStatus{
				AddressStatus: apis.AddressStatus{
					Address: &apis.Addressable{
						URL: brokerURL,
					},
				},
			},
		},
		&eventing.Broker{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Broker",
				APIVersion: eventing.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "not-ready",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "my-service",
			},
		},
	)
	require.NoError(t, err)

	brokerRef := corev1.ObjectReference{APIVersion: "eventing.knative.dev/v1", Kind: "Broker", Name: "default"}
	sinkURL, err := GetSinkURL(context.TODO(), client, &brokerRef, "my-ns")
	require.NoError(t, err)
	assert.Equal(t, "http://broker-ingress.knative-eventing.svc.cluster.local/my-ns/default", sinkURL.String())

	serviceRef := corev1.ObjectReference{APIVersion: "v1", Kind: "Service", Name: "my-service"}
	sinkURL, err = GetSinkURL(context.TODO(), client, &serviceRef, "my-ns")
	require.NoError(t, err)
	assert.Equal(t, "http://my-service.my-ns.svc/", sinkURL.String())

	notReadyRef := corev1.ObjectReference{APIVersion: "eventing.knative.dev/v1", Kind: "Broker", Name: "not-ready"}
	_, err = GetSinkURL(context.TODO(), client, &notReadyRef, "my-ns")
	require.ErrorContains(t, err, "does not contain address or URL")

	missingRef := corev1.ObjectReference{APIVersion: "eventing.knative.dev/v1", Kind: "Broker", Name: "missing"}
	_, err = GetSinkURL(context.TODO(), client, &missingRef, "my-ns")
	require.True(t, k8serrors.IsNotFound(err))
}
