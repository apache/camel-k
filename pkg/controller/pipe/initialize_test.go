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

package pipe

import (
	"context"
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	kameletsv1 "github.com/apache/camel-kamelets/crds/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"

	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewPipeError(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
	}
	c, err := internal.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.Error(t, err)
	assert.Equal(t, "no ref or URI specified in endpoint", err.Error())
	assert.Equal(t, v1.PipePhaseError, handledPipe.Status.Phase)
	cond := handledPipe.Status.GetCondition(v1.PipeConditionReady)
	assert.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, "IntegrationError", cond.Reason)
	assert.Equal(t, "no ref or URI specified in endpoint", cond.Message)
}

func TestNewPipeCamelURIBinding(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: ptr.To("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: ptr.To("log:info"),
			},
		},
	}
	c, err := internal.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	// Check integration which should have been created
	expectedIT := v1.NewIntegration(pipe.Namespace, pipe.Name)
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(&expectedIT), &expectedIT)
	require.NoError(t, err)
	assert.Equal(t, pipe.Name, expectedIT.Name)
	assert.Equal(t, v1.IntegrationPhaseNone, expectedIT.Status.Phase)
	assert.Equal(t, "Pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelKind])
	assert.Equal(t, "my-pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelName])
	flow, err := json.Marshal(expectedIT.Spec.Flows[0].RawMessage)
	require.NoError(t, err)
	assert.Equal(t, "{\"route\":{\"from\":{\"steps\":[{\"to\":\"log:info\"}],\"uri\":\"timer:tick\"},\"id\":\"binding\"}}", string(flow))
	// Verify icon propagation (nothing should be present), this is a value patched by the operator
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(pipe), pipe)
	require.NoError(t, err)
	assert.Equal(t, "", pipe.Annotations[kameletsv1.AnnotationIcon])
}

func TestNewPipe(t *testing.T) {
	source := kameletsv1.NewKamelet("ns", "my-source")
	source.Annotations = map[string]string{
		kameletsv1.AnnotationIcon: "my-source-icon-base64",
	}
	source.Spec = kameletsv1.KameletSpec{
		KameletSpecBase: kameletsv1.KameletSpecBase{
			Template: templateOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "timer:tick",
					"steps": []interface{}{
						map[string]interface{}{
							"to": "kamelet:sink",
						},
					},
				},
			}),
			Dependencies: []string{
				"camel:timer",
			},
		},
	}
	sink := kameletsv1.NewKamelet("ns", "my-sink")
	sink.Annotations = map[string]string{
		kameletsv1.AnnotationIcon: "my-sink-icon-base64",
	}
	sink.Spec = kameletsv1.KameletSpec{
		KameletSpecBase: kameletsv1.KameletSpecBase{
			Template: templateOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "kamelet:source",
					"steps": []interface{}{
						map[string]interface{}{
							"to": map[string]interface{}{
								"uri": "log:info",
							},
						},
					},
				},
			}),
			Dependencies: []string{
				"camel:log",
			},
		},
	}
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       kameletsv1.KameletKind,
					Namespace:  "ns",
					Name:       "my-source",
				},
			},
			Sink: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       kameletsv1.KameletKind,
					Namespace:  "ns",
					Name:       "my-sink",
				},
			},
		},
	}
	c, err := internal.NewFakeClient(pipe, &source, &sink)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	// Check integration which should have been created
	expectedIT := v1.NewIntegration(pipe.Namespace, pipe.Name)
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(&expectedIT), &expectedIT)
	require.NoError(t, err)
	assert.Equal(t, pipe.Name, expectedIT.Name)
	assert.Equal(t, v1.IntegrationPhaseNone, expectedIT.Status.Phase)
	assert.Equal(t, "Pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelKind])
	assert.Equal(t, "my-pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelName])
	flow, err := json.Marshal(expectedIT.Spec.Flows[0].RawMessage)
	require.NoError(t, err)
	assert.Equal(t, "{\"route\":{\"from\":{\"steps\":[{\"to\":\"kamelet:my-sink/sink\"}],\"uri\":\"kamelet:my-source/source\"},\"id\":\"binding\"}}", string(flow))
	// Verify icon propagation, this is a value patched by the operator
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(pipe), pipe)
	require.NoError(t, err)
	assert.Equal(t, "my-source-icon-base64", pipe.Annotations[kameletsv1.AnnotationIcon])
}

func templateOrFail(template map[string]interface{}) *kameletsv1.Template {
	data, err := json.Marshal(template)
	if err != nil {
		panic(err)
	}
	t := kameletsv1.Template{RawMessage: data}
	return &t
}

func TestNewPipeUnsupportedRef(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: ptr.To("timer:tick"),
			},
			Sink: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					APIVersion: "my-api-version",
					Kind:       "my-kind",
					Namespace:  "ns",
					Name:       "my-kind-name",
				},
			},
		},
	}
	c, err := internal.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.Error(t, err)
	assert.Equal(t, "could not find any suitable binding provider for my-api-version/my-kind my-kind-name in namespace ns. "+
		"Bindings available: [\"kamelet\" \"knative-uri\" \"strimzi\" \"service-ref\" \"camel-uri\" \"knative-ref\"]", err.Error())
	assert.Equal(t, v1.PipePhaseError, handledPipe.Status.Phase)
	cond := handledPipe.Status.GetCondition(v1.PipeConditionReady)
	assert.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, "IntegrationError", cond.Reason)
	assert.Equal(t, "could not find any suitable binding provider for my-api-version/my-kind my-kind-name in namespace ns. "+
		"Bindings available: [\"kamelet\" \"knative-uri\" \"strimzi\" \"service-ref\" \"camel-uri\" \"knative-ref\"]", cond.Message)
}

func TestNewPipeKnativeURIBinding(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Sink: v1.Endpoint{
				URI: ptr.To("http://my-knative-uri/"),
			},
			Source: v1.Endpoint{
				URI: ptr.To("direct:something"),
			},
		},
	}
	c, err := internal.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	// Check integration which should have been created
	expectedIT := v1.NewIntegration(pipe.Namespace, pipe.Name)
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(&expectedIT), &expectedIT)
	require.NoError(t, err)
	assert.Equal(t, pipe.Name, expectedIT.Name)
	assert.Equal(t, v1.IntegrationPhaseNone, expectedIT.Status.Phase)
	assert.Equal(t, "Pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelKind])
	assert.Equal(t, "my-pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelName])
	flow, err := json.Marshal(expectedIT.Spec.Flows[0].RawMessage)
	require.NoError(t, err)
	assert.Equal(t, "{\"route\":{\"from\":{\"steps\":[{\"to\":\"knative:endpoint/sink\"}],\"uri\":\"direct:something\"},\"id\":\"binding\"}}", string(flow))
	assert.Equal(t,
		"{\"services\":[{\"type\":\"endpoint\",\"name\":\"sink\",\"url\":\"http://my-knative-uri/\","+
			"\"metadata\":{\"camel.endpoint.kind\":\"sink\",\"knative.apiVersion\":\"\",\"knative.kind\":\"\",\"knative.name\":\"sink\"}}]}",
		expectedIT.Spec.Traits.Knative.Configuration,
	)
	assert.Equal(t, false, *expectedIT.Spec.Traits.Knative.SinkBinding)
}

func TestNewPipeKnativeRefBinding(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Sink: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Broker",
					Name:       "default",
					APIVersion: "eventing.knative.dev/v1",
				},
			},
			Source: v1.Endpoint{
				URI: ptr.To("direct:something"),
			},
		},
	}
	c, err := internal.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	// Check integration which should have been created
	expectedIT := v1.NewIntegration(pipe.Namespace, pipe.Name)
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(&expectedIT), &expectedIT)
	require.NoError(t, err)
	assert.Equal(t, pipe.Name, expectedIT.Name)
	assert.Equal(t, v1.IntegrationPhaseNone, expectedIT.Status.Phase)
	assert.Equal(t, "Pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelKind])
	assert.Equal(t, "my-pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelName])
	flow, err := json.Marshal(expectedIT.Spec.Flows[0].RawMessage)
	require.NoError(t, err)
	assert.Equal(t, "{\"route\":{\"from\":{\"steps\":[{\"to\":\"knative:event?apiVersion=eventing.knative.dev%2Fv1\\u0026kind=Broker\\u0026name=default\"}],"+
		"\"uri\":\"direct:something\"},\"id\":\"binding\"}}", string(flow))
}

func TestNewPipeStrimziKafkaTopicBinding(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Sink: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "KafkaTopic",
					Name:       "mytopic",
					APIVersion: "kafka.strimzi.io/v1beta2",
				},
				Properties: asEndpointProperties(map[string]string{
					"brokers": "my-cluster-kafka-bootstrap:9092",
				}),
			},
			Source: v1.Endpoint{
				URI: ptr.To("direct:something"),
			},
		},
	}
	c, err := internal.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	// Check integration which should have been created
	expectedIT := v1.NewIntegration(pipe.Namespace, pipe.Name)
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(&expectedIT), &expectedIT)
	require.NoError(t, err)
	assert.Equal(t, pipe.Name, expectedIT.Name)
	assert.Equal(t, v1.IntegrationPhaseNone, expectedIT.Status.Phase)
	assert.Equal(t, "Pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelKind])
	assert.Equal(t, "my-pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelName])
	flow, err := json.Marshal(expectedIT.Spec.Flows[0].RawMessage)
	require.NoError(t, err)
	assert.Equal(t, "{\"route\":{\"from\":{\"steps\":[{\"to\":\"kafka:mytopic?brokers=my-cluster-kafka-bootstrap%3A9092\"}],"+
		"\"uri\":\"direct:something\"},\"id\":\"binding\"}}", string(flow))
}

func TestNewPipeStrimziKafkaBinding(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Sink: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kafka",
					Name:       "my-kafka",
					APIVersion: "kafka.strimzi.io/v1beta2",
				},
				Properties: asEndpointProperties(map[string]string{
					"topic":   "mytopic",
					"brokers": "my-cluster-kafka-bootstrap:9092",
				}),
			},
			Source: v1.Endpoint{
				URI: ptr.To("direct:something"),
			},
		},
	}
	c, err := internal.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	// Check integration which should have been created
	expectedIT := v1.NewIntegration(pipe.Namespace, pipe.Name)
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(&expectedIT), &expectedIT)
	require.NoError(t, err)
	assert.Equal(t, pipe.Name, expectedIT.Name)
	assert.Equal(t, v1.IntegrationPhaseNone, expectedIT.Status.Phase)
	assert.Equal(t, "Pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelKind])
	assert.Equal(t, "my-pipe", expectedIT.Labels[kubernetes.CamelCreatorLabelName])
	flow, err := json.Marshal(expectedIT.Spec.Flows[0].RawMessage)
	require.NoError(t, err)
	assert.Equal(t, "{\"route\":{\"from\":{\"steps\":[{\"to\":\"kafka:mytopic?brokers=my-cluster-kafka-bootstrap%3A9092\"}],"+
		"\"uri\":\"direct:something\"},\"id\":\"binding\"}}", string(flow))
}

func asEndpointProperties(props map[string]string) *v1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &v1.EndpointProperties{
		RawMessage: serialized,
	}
}
