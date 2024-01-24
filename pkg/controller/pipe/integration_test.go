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
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/dsl"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateIntegrationForPipe(t *testing.T) {
	client, err := test.NewFakeClient()
	assert.NoError(t, err)

	pipe := nominalPipe("my-pipe")
	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	assert.Nil(t, err)
	assert.Equal(t, "my-pipe", it.Name)
	assert.Equal(t, "default", it.Namespace)
	assert.Equal(t, map[string]string{
		"my-annotation": "my-annotation-val",
	}, it.Annotations)
	assert.Equal(t, map[string]string{
		"camel.apache.org/created.by.kind": "Pipe",
		"camel.apache.org/created.by.name": "my-pipe",
		"my-label":                         "my-label-val",
	}, it.Labels)
	assert.Equal(t, "camel.apache.org/v1", it.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Pipe", it.OwnerReferences[0].Kind)
	assert.Equal(t, "my-pipe", it.OwnerReferences[0].Name)
	dsl, err := dsl.ToYamlDSL(it.Spec.Flows)
	assert.Nil(t, err)
	assert.Equal(t, expectedNominalRoute(), string(dsl))
}

func TestCreateIntegrationForPipeDataType(t *testing.T) {
	client, err := test.NewFakeClient()
	assert.NoError(t, err)

	pipe := nominalPipe("my-pipe-data-type")
	pipe.Spec.Sink.DataTypes = map[v1.TypeSlot]v1.DataTypeReference{
		v1.TypeSlotIn: {
			Format: "string",
		},
	}
	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	assert.Nil(t, err)
	dsl, err := dsl.ToYamlDSL(it.Spec.Flows)
	assert.Nil(t, err)
	assert.Equal(t, expectedNominalRouteWithDataType("data-type-action"), string(dsl))
}

func TestCreateIntegrationForPipeDataTypeOverridden(t *testing.T) {
	client, err := test.NewFakeClient()
	assert.NoError(t, err)

	pipe := nominalPipe("my-pipe-data-type")
	pipe.Spec.Sink.DataTypes = map[v1.TypeSlot]v1.DataTypeReference{
		v1.TypeSlotIn: {
			Format: "string",
		},
	}
	newDataTypeKameletAction := "data-type-action-v4-2"
	pipe.Annotations[v1.KameletDataTypeLabel] = newDataTypeKameletAction
	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	assert.Nil(t, err)
	dsl, err := dsl.ToYamlDSL(it.Spec.Flows)
	assert.Nil(t, err)
	assert.Equal(t, expectedNominalRouteWithDataType(newDataTypeKameletAction), string(dsl))
}

func nominalPipe(name string) v1.Pipe {
	pipe := v1.NewPipe("default", name)
	pipe.Annotations = map[string]string{
		"my-annotation": "my-annotation-val",
	}
	pipe.Labels = map[string]string{
		"my-label": "my-label-val",
	}
	pipe.Spec.Source = v1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "Kamelet",
			Name:       "my-source",
			APIVersion: "camel.apache.org/v1",
		},
	}
	pipe.Spec.Sink = v1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "Kamelet",
			Name:       "my-sink",
			APIVersion: "camel.apache.org/v1",
		},
	}
	pipe.Status.Phase = v1.PipePhaseReady
	return pipe
}

func expectedNominalRoute() string {
	return `- route:
    from:
      steps:
      - to: kamelet:my-sink/sink
      uri: kamelet:my-source/source
    id: binding
`
}

func expectedNominalRouteWithDataType(name string) string {
	return `- route:
    from:
      steps:
      - kamelet:
          name: ` + name + `/sink-in
      - to: kamelet:my-sink/sink
      uri: kamelet:my-source/source
    id: binding
`
}
