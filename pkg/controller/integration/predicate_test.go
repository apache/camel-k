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

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
)

func newKnativeServiceWithStatus(t *testing.T, url string, ready metav1.ConditionStatus) *servingv1.Service {
	t.Helper()
	parsed, err := apis.ParseURL(url)
	require.NoError(t, err)

	return &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-ksvc",
		},
		Status: servingv1.ServiceStatus{
			Status: apis.Status{
				Conditions: []metav1.Condition{
					{Type: servingv1.ServiceConditionReady, Status: ready},
				},
			},
			RouteStatusFields: servingv1.RouteStatusFields{
				URL:     parsed,
				Address: &apis.Addressable{URL: parsed},
			},
		},
	}
}

func TestStatusChangedPredicateKnativeService(t *testing.T) {
	p := StatusChangedPredicate{}

	// A URL with user information makes url.URL carry an unexported *url.Userinfo:
	// the semantic equality can only compare it through the URL equality function.
	old := newKnativeServiceWithStatus(t, "http://user:pass@my-ksvc.ns.svc.cluster.local", metav1.ConditionTrue)

	same := old.DeepCopy()
	same.Generation++
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: same}))

	changedURL := newKnativeServiceWithStatus(t, "http://user:pass@my-ksvc.ns.svc.cluster.local/other", metav1.ConditionTrue)
	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: changedURL}))

	notReady := newKnativeServiceWithStatus(t, "http://user:pass@my-ksvc.ns.svc.cluster.local", metav1.ConditionFalse)
	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: notReady}))

	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: nil, ObjectNew: same}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nil}))
}
