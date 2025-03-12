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

package v1

import (
	context "context"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	applyconfigurationcamelv1 "github.com/apache/camel-k/v2/pkg/client/camel/applyconfiguration/camel/v1"
	scheme "github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned/scheme"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// PipesGetter has a method to return a PipeInterface.
// A group's client should implement this interface.
type PipesGetter interface {
	Pipes(namespace string) PipeInterface
}

// PipeInterface has methods to work with Pipe resources.
type PipeInterface interface {
	Create(ctx context.Context, pipe *camelv1.Pipe, opts metav1.CreateOptions) (*camelv1.Pipe, error)
	Update(ctx context.Context, pipe *camelv1.Pipe, opts metav1.UpdateOptions) (*camelv1.Pipe, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, pipe *camelv1.Pipe, opts metav1.UpdateOptions) (*camelv1.Pipe, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*camelv1.Pipe, error)
	List(ctx context.Context, opts metav1.ListOptions) (*camelv1.PipeList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *camelv1.Pipe, err error)
	Apply(ctx context.Context, pipe *applyconfigurationcamelv1.PipeApplyConfiguration, opts metav1.ApplyOptions) (result *camelv1.Pipe, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, pipe *applyconfigurationcamelv1.PipeApplyConfiguration, opts metav1.ApplyOptions) (result *camelv1.Pipe, err error)
	GetScale(ctx context.Context, pipeName string, options metav1.GetOptions) (*autoscalingv1.Scale, error)
	UpdateScale(ctx context.Context, pipeName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error)

	PipeExpansion
}

// pipes implements PipeInterface
type pipes struct {
	*gentype.ClientWithListAndApply[*camelv1.Pipe, *camelv1.PipeList, *applyconfigurationcamelv1.PipeApplyConfiguration]
}

// newPipes returns a Pipes
func newPipes(c *CamelV1Client, namespace string) *pipes {
	return &pipes{
		gentype.NewClientWithListAndApply[*camelv1.Pipe, *camelv1.PipeList, *applyconfigurationcamelv1.PipeApplyConfiguration](
			"pipes",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *camelv1.Pipe { return &camelv1.Pipe{} },
			func() *camelv1.PipeList { return &camelv1.PipeList{} },
		),
	}
}

// GetScale takes name of the pipe, and returns the corresponding autoscalingv1.Scale object, and an error if there is any.
func (c *pipes) GetScale(ctx context.Context, pipeName string, options metav1.GetOptions) (result *autoscalingv1.Scale, err error) {
	result = &autoscalingv1.Scale{}
	err = c.GetClient().Get().
		Namespace(c.GetNamespace()).
		Resource("pipes").
		Name(pipeName).
		SubResource("scale").
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// UpdateScale takes the top resource name and the representation of a scale and updates it. Returns the server's representation of the scale, and an error, if there is any.
func (c *pipes) UpdateScale(ctx context.Context, pipeName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (result *autoscalingv1.Scale, err error) {
	result = &autoscalingv1.Scale{}
	err = c.GetClient().Put().
		Namespace(c.GetNamespace()).
		Resource("pipes").
		Name(pipeName).
		SubResource("scale").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(scale).
		Do(ctx).
		Into(result)
	return
}
