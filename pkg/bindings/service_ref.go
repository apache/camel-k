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
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

// ServiceRefBindingProvider converts a Service into a Camel http endpoint.
type ServiceRefBindingProvider struct{}

// ID --.
func (k ServiceRefBindingProvider) ID() string {
	return "service-ref"
}

// Translate will check the endpoint reference being either a Service or an Integration and Pipe which have a Service associated.
func (k ServiceRefBindingProvider) Translate(ctx BindingContext, endpointCtx EndpointContext, e v1.Endpoint) (*Binding, error) {
	if e.Ref == nil || (!isService(e.Ref) && !isIntegration(e.Ref) && !isPipe(e.Ref)) {
		// works only on Service refs
		return nil, nil
	}

	// IMPORTANT: when we have a Service associated to either a Pipe or an Integration,
	// this Service has the same namespace and the same name of the resource, so, we can lookup with those info
	namespace := e.Ref.Namespace
	if namespace == "" {
		namespace = ctx.Namespace
	}
	svc, err := kubernetes.LookupService(ctx.Ctx, ctx.Client, namespace, e.Ref.Name)
	if err != nil {
		return nil, err
	}
	if svc == nil {
		return nil, fmt.Errorf("could not load a Service with name %s in namespace %s", e.Ref.Name, namespace)
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		return nil, fmt.Errorf(
			"operator only supports %s Service type, feel free to request this support",
			corev1.ServiceTypeClusterIP,
		)
	}

	svcURI := kubernetes.GetClusterTypeServiceURI(svc)
	props, err := e.Properties.GetPropertyMap()
	if err != nil {
		return nil, err
	}
	if props["path"] != "" {
		if !strings.HasPrefix(props["path"], "/") {
			svcURI += "/"
		}
		svcURI += props["path"]
	}

	return &Binding{
		URI: svcURI,
	}, nil
}

func isService(ref *corev1.ObjectReference) bool {
	return ref.APIVersion == corev1.SchemeGroupVersion.String() && ref.Kind == "Service"
}

func isIntegration(ref *corev1.ObjectReference) bool {
	return ref.APIVersion == v1.SchemeGroupVersion.String() && ref.Kind == v1.IntegrationKind
}

func isPipe(ref *corev1.ObjectReference) bool {
	return ref.APIVersion == v1.SchemeGroupVersion.String() && ref.Kind == v1.PipeKind
}

// Order --.
//
//nolint:mnd
func (k ServiceRefBindingProvider) Order() int {
	return OrderLast - 10
}

func init() {
	RegisterBindingProvider(ServiceRefBindingProvider{})
}
