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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
)

const (
	OrderFirst    = 0
	OrderStandard = 50
	OrderLast     = 100
)

// Binding represents how a Kubernetes object is represented in Camel K resources
type Binding struct {
	// URI is the Camel URI equivalent
	URI string
	// Traits is a partial trait specification that should be merged into the integration
	Traits map[string]v1.TraitSpec
	// ApplicationProperties contain properties that should be set on the integration for the binding to work
	ApplicationProperties map[string]string
}

// BindingProvider maps a KameletBinding endpoint into Camel K resources
type BindingProvider interface {
	// ID returns the name of the binding provider
	ID() string
	// Translate does the actual mapping
	Translate(ctx BindingContext, endpointType v1alpha1.EndpointType, endpoint v1alpha1.Endpoint) (*Binding, error)
	// Order returns the relative order of execution of the binding provider
	Order() int
}

type BindingContext struct {
	Ctx       context.Context
	Client    client.Client
	Namespace string
	Profile   v1.TraitProfile
}
