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

package kameletbinding

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"

	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKLBUnsupportedRef(t *testing.T) {
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-svc",
			Namespace: "ns",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{},
			Selector: map[string]string{
				v1.IntegrationLabel: "my-klb",
			},
		},
	}
	klb := &v1alpha1.KameletBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.KameletBindingKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-klb",
		},
		Spec: v1alpha1.KameletBindingSpec{
			Source: v1alpha1.Endpoint{
				URI: ptr.To("timer:tick"),
			},
			Sink: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					APIVersion: svc.APIVersion,
					Kind:       svc.Kind,
					Namespace:  svc.Namespace,
					Name:       svc.Name,
				},
			},
		},
	}
	c, err := test.NewFakeClient(klb)
	require.NoError(t, err)

	a := NewInitializeAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(klb))
	handledKlb, err := a.Handle(context.TODO(), klb)
	require.Error(t, err)
	assert.Equal(t, "could not find any suitable binding provider for v1/Service my-svc in namespace ns. "+
		"Bindings available: [\"kamelet\" \"knative-uri\" \"strimzi\" \"camel-uri\" \"knative-ref\"]", err.Error())
	assert.Equal(t, v1alpha1.KameletBindingPhaseError, handledKlb.Status.Phase)
	cond := handledKlb.Status.GetCondition(v1alpha1.KameletBindingIntegrationConditionError)
	assert.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, "could not find any suitable binding provider for v1/Service my-svc in namespace ns. "+
		"Bindings available: [\"kamelet\" \"knative-uri\" \"strimzi\" \"camel-uri\" \"knative-ref\"]", cond.Message)
}
