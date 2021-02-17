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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

var bindingProviders []BindingProvider

func RegisterBindingProvider(bp BindingProvider) {
	bindingProviders = append(bindingProviders, bp)
	sort.Slice(bindingProviders, func(i, j int) bool {
		bi := bindingProviders[i]
		bj := bindingProviders[j]
		return (bi.Order() < bj.Order()) ||
			(bi.Order() == bj.Order() && bi.ID() < bj.ID())
	})
}

// Translate execute all chained binding providers, returning the first success or the first error
func Translate(ctx BindingContext, endpointCtx EndpointContext, endpoint v1alpha1.Endpoint) (*Binding, error) {
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

func validateEndpoint(ctx BindingContext, e v1alpha1.Endpoint) error {
	if e.Ref == nil && e.URI == nil {
		return errors.New("no ref or URI specified in endpoint")
	} else if e.Ref != nil && e.URI != nil {
		return errors.New("cannot use both ref and URI to specify an endpoint: only one of them should be used")
	}
	if e.Ref != nil && e.Ref.Namespace != "" && e.Ref.Namespace != ctx.Namespace {
		return errors.New("cross-namespace references are not allowed in kamelet binding")
	}
	return nil
}
