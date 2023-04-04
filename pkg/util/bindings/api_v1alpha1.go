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

// Package bindings provides APIs to transform Kubernetes objects into Camel URIs equivalents
package bindings

import (
	"context"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	v1alpha1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/client"
)

// V1alpha1BindingProvider maps a Binding endpoint into Camel K resources.
// Deprecated.
type V1alpha1BindingProvider interface {
	// ID returns the name of the binding provider
	ID() string
	// Translate does the actual mapping
	Translate(ctx V1alpha1BindingContext, endpointContext V1alpha1EndpointContext, endpoint v1alpha1.Endpoint) (*Binding, error)
	// Order returns the relative order of execution of the binding provider
	Order() int
}

// V1alpha1BindingContext --
// nolint: containedctx
// Deprecated.
type V1alpha1BindingContext struct {
	Ctx       context.Context
	Client    client.Client
	Namespace string
	Profile   v1.TraitProfile
}

// V1alpha1EndpointContext --
// Deprecated.
type V1alpha1EndpointContext struct {
	Type     v1alpha1.EndpointType
	Position *int
}
