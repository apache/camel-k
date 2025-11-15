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
	"fmt"
	"sort"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"k8s.io/utils/ptr"
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

// Translate execute all chained binding providers, returning the first success or the first error.
func Translate(ctx BindingContext, endpointCtx EndpointContext, endpoint v1.Endpoint) (*Binding, error) {
	availableBindings := make([]string, len(bindingProviders))
	if err := validateEndpoint(ctx, endpoint); err != nil {
		return nil, err
	}

	for i, bp := range bindingProviders {
		availableBindings[i] = bp.ID()
		b, err := bp.Translate(ctx, endpointCtx, endpoint)
		if b != nil || err != nil {
			return b, err
		}
	}

	// If no success we return an error with the actual list of available binding providers
	var errorMessage string
	if endpoint.Ref != nil {
		errorMessage = fmt.Sprintf("could not find any suitable binding provider for %s/%s %s in namespace %s. Bindings available: %q",
			endpoint.Ref.APIVersion, endpoint.Ref.Kind, endpoint.Ref.Name, endpoint.Ref.Namespace, availableBindings)
	} else if ptr.Deref(endpoint.URI, "") != "" {
		errorMessage = fmt.Sprintf("could not find any suitable binding provider for %s", *endpoint.URI)
	}
	return nil, errors.New(errorMessage)
}

func validateEndpoint(ctx BindingContext, e v1.Endpoint) error {
	if e.Ref == nil && e.URI == nil {
		return errors.New("no ref or URI specified in endpoint")
	} else if e.Ref != nil && e.URI != nil {
		return errors.New("cannot use both ref and URI to specify an endpoint: only one of them should be used")
	}
	if e.Ref != nil && e.Ref.Namespace != "" && e.Ref.Namespace != ctx.Namespace {
		if ok, err := isKnownKnativeResource(e.Ref); ok {
			if err != nil {
				return err
			}
			return errors.New("cross-namespace Pipe references are not allowed for Knative")
		}
		// only check this when there is a cross-namespace access
		return verifyResourceRBAC(ctx, e)
	}

	return nil
}

// verifyResourceRBAC verify if the ServiceAccount which is running the Integration has enough
// privileges to access the resource. We avoid any potential cross-namespace vulnerability access.
func verifyResourceRBAC(ctx BindingContext, e v1.Endpoint) error {
	resources := strings.ToLower(e.Ref.Kind) + "s"
	if ctx.ServiceAccountName == "" {
		return fmt.Errorf("you must to use an authorized ServiceAccount to access cross-namespace resources %s. "+
			"Set it in the Pipe spec accordingly", resources)
	}
	ok, err := kubernetes.CheckServiceAccountPermission(
		ctx.Ctx,
		ctx.Client,
		fmt.Sprintf("system:serviceaccount:%s:%s", ctx.Namespace, ctx.ServiceAccountName),
		e.Ref.GroupVersionKind().Group,
		resources,
		e.Ref.Namespace,
		"get",
	)

	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("cross-namespace Pipe reference authorization denied for the ServiceAccount %s and resources %s",
			ctx.ServiceAccountName, resources)
	}
	return nil
}
