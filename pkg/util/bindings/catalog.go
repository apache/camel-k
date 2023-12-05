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

package bindings

import (
	"errors"
	"sort"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/platform"
)

var bindingProviders []BindingProvider

// RegisterBindingProvider --.
func RegisterBindingProvider(bp BindingProvider) {
	bindingProviders = append(bindingProviders, bp)
	sort.Slice(bindingProviders, func(i, j int) bool {
		bi := bindingProviders[i]
		bj := bindingProviders[j]
		return (bi.Order() < bj.Order()) ||
			(bi.Order() == bj.Order() && bi.ID() < bj.ID())
	})
}

// Deprecated.
var v1alpha1BindingProviders []V1alpha1BindingProvider

// V1alpha1RegisterBindingProvider --
// Deprecated.
func V1alpha1RegisterBindingProvider(bp V1alpha1BindingProvider) {
	v1alpha1BindingProviders = append(v1alpha1BindingProviders, bp)
	sort.Slice(v1alpha1BindingProviders, func(i, j int) bool {
		bi := v1alpha1BindingProviders[i]
		bj := v1alpha1BindingProviders[j]
		return (bi.Order() < bj.Order()) ||
			(bi.Order() == bj.Order() && bi.ID() < bj.ID())
	})
}

// Translate execute all chained binding providers, returning the first success or the first error.
func Translate(ctx BindingContext, endpointCtx EndpointContext, endpoint v1.Endpoint) (*Binding, error) {
	if err := validateEndpoint(ctx, endpoint); err != nil {
		return nil, err
	}

	for _, bp := range bindingProviders {
		b, err := bp.Translate(ctx, endpointCtx, endpoint)
		if b != nil || err != nil {
			return b, err
		}
	}
	return nil, nil
}

func validateEndpoint(ctx BindingContext, e v1.Endpoint) error {
	if e.Ref == nil && e.URI == nil {
		return errors.New("no ref or URI specified in endpoint")
	} else if e.Ref != nil && e.URI != nil {
		return errors.New("cannot use both ref and URI to specify an endpoint: only one of them should be used")
	}
	if e.Ref != nil && e.Ref.Namespace != "" && e.Ref.Namespace != ctx.Namespace {
		// referencing default Kamelets in operator namespace is allowed
		if e.Ref.Kind == v1.KameletKind && e.Ref.Namespace == platform.GetOperatorNamespace() {
			return nil
		}

		return errors.New("cross-namespace references are not allowed in Pipe")
	}
	return nil
}

// TranslateV1alpha1 execute all chained binding providers, returning the first success or the first error.
// Deprecated.
func TranslateV1alpha1(ctx V1alpha1BindingContext, endpointCtx V1alpha1EndpointContext, endpoint v1alpha1.Endpoint) (*Binding, error) {
	if err := validateEndpointV1alpha1(ctx, endpoint); err != nil {
		return nil, err
	}

	for _, bp := range v1alpha1BindingProviders {
		b, err := bp.Translate(ctx, endpointCtx, endpoint)
		if b != nil || err != nil {
			return b, err
		}
	}
	return nil, nil
}

// Deprecated.
func validateEndpointV1alpha1(ctx V1alpha1BindingContext, e v1alpha1.Endpoint) error {
	if e.Ref == nil && e.URI == nil {
		return errors.New("no ref or URI specified in endpoint")
	} else if e.Ref != nil && e.URI != nil {
		return errors.New("cannot use both ref and URI to specify an endpoint: only one of them should be used")
	}
	if e.Ref != nil && e.Ref.Namespace != "" && e.Ref.Namespace != ctx.Namespace {
		// referencing default Kamelets in operator namespace is allowed
		if e.Ref.Kind == v1.KameletKind && e.Ref.Namespace == platform.GetOperatorNamespace() {
			return nil
		}

		return errors.New("cross-namespace references are not allowed in KameletBinding")
	}
	return nil
}
