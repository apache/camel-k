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
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestCreateIntegrationForPipe(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-pipe")
	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
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
	dsl, err := v1.ToYamlDSL(it.Spec.Flows)
	require.NoError(t, err)
	assert.Equal(t, expectedNominalRoute(), string(dsl))
}

func TestCreateIntegrationForPipeWithSinkKameletErrorHandler(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-error-handler-pipe")
	pipe.Spec.ErrorHandler = &v1.ErrorHandlerSpec{
		RawMessage: []byte(`{"sink": {"endpoint": {"ref": {"kind": "Kamelet", "apiVersion": "camel.apache.org/v1", "name": "my-err"}}}}`),
	}

	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
	assert.Equal(t, "my-error-handler-pipe", it.Name)
	assert.Equal(t, "default", it.Namespace)
	assert.Equal(t, "camel.apache.org/v1", it.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Pipe", it.OwnerReferences[0].Kind)
	assert.Equal(t, "my-error-handler-pipe", it.OwnerReferences[0].Name)
	dsl, err := v1.ToYamlDSL(it.Spec.Flows)
	require.NoError(t, err)
	assert.Equal(t,
		`- errorHandler:
    deadLetterChannel:
      deadLetterUri: kamelet:my-err/errorHandler
- route:
    from:
      steps:
      - to: kamelet:my-sink/sink
      uri: kamelet:my-source/source
    id: binding
`, string(dsl),
	)
}

func TestCreateIntegrationForPipeWithSinkErrorHandler(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-error-handler-pipe")
	pipe.Spec.ErrorHandler = &v1.ErrorHandlerSpec{
		RawMessage: []byte(`{"sink": {"endpoint": {"uri": "someUri"}}}`),
	}

	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
	assert.Equal(t, "my-error-handler-pipe", it.Name)
	assert.Equal(t, "default", it.Namespace)
	assert.Equal(t, "camel.apache.org/v1", it.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Pipe", it.OwnerReferences[0].Kind)
	assert.Equal(t, "my-error-handler-pipe", it.OwnerReferences[0].Name)
	dsl, err := v1.ToYamlDSL(it.Spec.Flows)
	require.NoError(t, err)
	assert.Equal(t,
		`- errorHandler:
    deadLetterChannel:
      deadLetterUri: someUri
- route:
    from:
      steps:
      - to: kamelet:my-sink/sink
      uri: kamelet:my-source/source
    id: binding
`, string(dsl),
	)
}

func TestCreateIntegrationForPipeWithLogErrorHandler(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-error-handler-pipe")
	pipe.Spec.ErrorHandler = &v1.ErrorHandlerSpec{
		RawMessage: []byte(`{"log": {"parameters": {"showHeaders": "true"}}}`),
	}

	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
	assert.Equal(t, "my-error-handler-pipe", it.Name)
	assert.Equal(t, "default", it.Namespace)
	assert.Equal(t, "camel.apache.org/v1", it.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Pipe", it.OwnerReferences[0].Kind)
	assert.Equal(t, "my-error-handler-pipe", it.OwnerReferences[0].Name)
	dsl, err := v1.ToYamlDSL(it.Spec.Flows)
	require.NoError(t, err)
	assert.Equal(t,
		`- errorHandler:
    defaultErrorHandler:
      logName: err
- route:
    from:
      steps:
      - to: kamelet:my-sink/sink
      uri: kamelet:my-source/source
    id: binding
`, string(dsl),
	)
}

func TestCreateIntegrationForPipeWithNoneErrorHandler(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-error-handler-pipe")
	pipe.Spec.ErrorHandler = &v1.ErrorHandlerSpec{
		RawMessage: []byte(`{"none": {}}`),
	}

	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
	assert.Equal(t, "my-error-handler-pipe", it.Name)
	assert.Equal(t, "default", it.Namespace)
	assert.Equal(t, "camel.apache.org/v1", it.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Pipe", it.OwnerReferences[0].Kind)
	assert.Equal(t, "my-error-handler-pipe", it.OwnerReferences[0].Name)
	dsl, err := v1.ToYamlDSL(it.Spec.Flows)
	require.NoError(t, err)
	assert.Equal(t,
		`- errorHandler:
    noErrorHandler: {}
- route:
    from:
      steps:
      - to: kamelet:my-sink/sink
      uri: kamelet:my-source/source
    id: binding
`,
		string(dsl),
	)
}

func TestCreateIntegrationForPipeDataType(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-pipe-data-type")
	pipe.Spec.Sink.DataTypes = map[v1.TypeSlot]v1.DataTypeReference{
		v1.TypeSlotIn: {
			Format: "string",
		},
	}
	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
	dsl, err := v1.ToYamlDSL(it.Spec.Flows)
	require.NoError(t, err)
	assert.Equal(t, expectedNominalRouteWithDataType("data-type-action"), string(dsl))
}

func TestCreateIntegrationForPipeDataTypeOverridden(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-pipe-data-type")
	pipe.Spec.Sink.DataTypes = map[v1.TypeSlot]v1.DataTypeReference{
		v1.TypeSlotIn: {
			Format: "string",
		},
	}
	newDataTypeKameletAction := "data-type-action-v4-2"
	pipe.Annotations[v1.KameletDataTypeLabel] = newDataTypeKameletAction
	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
	dsl, err := v1.ToYamlDSL(it.Spec.Flows)
	require.NoError(t, err)
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

func TestExtractTraitAnnotations(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	annotations := map[string]string{
		"my-personal-annotation":                                 "hello",
		v1.TraitAnnotationPrefix + "service.enabled":             "true",
		v1.TraitAnnotationPrefix + "container.image-pull-policy": "Never",
		v1.TraitAnnotationPrefix + "camel.runtime-version":       "1.2.3",
		v1.TraitAnnotationPrefix + "camel.properties":            `["prop1=1", "prop2=2"]`,
		v1.TraitAnnotationPrefix + "environment.vars":            `["env1=1"]`,
	}
	traits, err := extractAndDeleteTraits(client, annotations)
	require.NoError(t, err)
	assert.Equal(t, ptr.To(true), traits.Service.Enabled)
	assert.Equal(t, corev1.PullNever, traits.Container.ImagePullPolicy)
	assert.Equal(t, "1.2.3", traits.Camel.RuntimeVersion)
	assert.Equal(t, []string{"prop1=1", "prop2=2"}, traits.Camel.Properties)
	assert.Equal(t, []string{"env1=1"}, traits.Environment.Vars)
	assert.Len(t, annotations, 1)
	assert.Empty(t, annotations[v1.TraitAnnotationPrefix+"service.enabled"])
	assert.Empty(t, annotations[v1.TraitAnnotationPrefix+"container.image-pull-policy"])
	assert.Empty(t, annotations[v1.TraitAnnotationPrefix+"camel.runtime-version"])
	assert.Empty(t, annotations[v1.TraitAnnotationPrefix+"camel.properties"])
	assert.Empty(t, annotations[v1.TraitAnnotationPrefix+"environment.vars"])
	assert.Equal(t, "hello", annotations["my-personal-annotation"])
}

func TestExtractTraitAnnotationsError(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	annotations := map[string]string{
		"my-personal-annotation":                       "hello",
		v1.TraitAnnotationPrefix + "servicefake.bogus": "true",
	}
	traits, err := extractAndDeleteTraits(client, annotations)
	require.Error(t, err)
	assert.Equal(t, "trait servicefake does not exist in catalog", err.Error())
	assert.Nil(t, traits)
	assert.Len(t, annotations, 2)
}

func TestExtractTraitAnnotationsEmpty(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	annotations := map[string]string{
		"my-personal-annotation": "hello",
	}
	traits, err := extractAndDeleteTraits(client, annotations)
	require.NoError(t, err)
	assert.Nil(t, traits)
	assert.Len(t, annotations, 1)
}

func TestCreateIntegrationTraitsForPipeWithTraitAnnotations(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	pipe := nominalPipe("my-pipe")
	pipe.Annotations[v1.TraitAnnotationPrefix+"service.enabled"] = "true"

	it, err := CreateIntegrationFor(context.TODO(), client, &pipe)
	require.NoError(t, err)
	assert.Equal(t, "my-pipe", it.Name)
	assert.Equal(t, "default", it.Namespace)
	assert.Equal(t, map[string]string{
		"my-annotation": "my-annotation-val",
	}, it.Annotations)
	assert.Equal(t, ptr.To(true), it.Spec.Traits.Service.Enabled)
}
