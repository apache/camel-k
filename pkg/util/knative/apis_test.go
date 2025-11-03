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

	"github.com/apache/camel-k/v2/pkg/internal/knative"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestAPIs(t *testing.T) {
	ref, err := ExtractObjectReference("knative:endpoint/ciao")
	require.NoError(t, err)
	refs := FillMissingReferenceData(knative.CamelServiceTypeEndpoint, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Service",
		APIVersion: "serving.knative.dev/v1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:endpoint/ciao?apiVersion=serving.knative.dev/v1")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEndpoint, ref)
	checkValidRefs(t, refs)
	assert.Len(t, refs, 1)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Service",
		APIVersion: "serving.knative.dev/v1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:endpoint/ciao?apiVersion=serving.knative.dev/v1&kind=Xxx")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEndpoint, ref)
	checkValidRefs(t, refs)
	assert.Len(t, refs, 1)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Xxx",
		APIVersion: "serving.knative.dev/v1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:endpoint/ciao?apiVersion=yyy&kind=Xxx")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEndpoint, ref)
	checkValidRefs(t, refs)
	assert.Len(t, refs, 1)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Xxx",
		APIVersion: "yyy",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:endpoint/ciao?kind=Service")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEndpoint, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Service",
		APIVersion: "serving.knative.dev/v1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:endpoint/ciao?kind=Channel")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEndpoint, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Channel",
		APIVersion: "messaging.knative.dev/v1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:endpoint/ciao?kind=KafkaChannel")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEndpoint, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "KafkaChannel",
		APIVersion: "messaging.knative.dev/v1beta1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:channel/ciao")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeChannel, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Channel",
		APIVersion: "messaging.knative.dev/v1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:channel/ciao?apiVersion=messaging.knative.dev/v1")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeChannel, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Channel",
		APIVersion: "messaging.knative.dev/v1",
		Name:       "ciao",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:channel/ciao?apiVersion=xxx.knative.dev/v1alpha1")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeChannel, ref)
	assert.Empty(t, refs)

	ref, err = ExtractObjectReference("knative:event/ciao")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEvent, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Broker",
		APIVersion: "eventing.knative.dev/v1",
		Name:       "default",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:event/ciao?apiVersion=xxx")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEvent, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Broker",
		APIVersion: "xxx",
		Name:       "default",
	}, refs[0])

	ref, err = ExtractObjectReference("knative:event/ciao?name=aaa")
	require.NoError(t, err)
	refs = FillMissingReferenceData(knative.CamelServiceTypeEvent, ref)
	checkValidRefs(t, refs)
	assert.Equal(t, v1.ObjectReference{
		Kind:       "Broker",
		APIVersion: "eventing.knative.dev/v1",
		Name:       "aaa",
	}, refs[0])
}

func checkValidRefs(t *testing.T, refs []v1.ObjectReference) {
	t.Helper()

	assert.Greater(t, len(refs), 0)
	for _, ref := range refs {
		assert.NotNil(t, ref.Name)
		assert.NotNil(t, ref.Kind)
		assert.NotNil(t, ref.APIVersion)
	}
}
