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

package apis

import (
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	eventingv1beta1 "knative.dev/eventing/pkg/apis/eventing/v1beta1"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	sourcesv1beta2 "knative.dev/eventing/pkg/apis/sources/v1beta2"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, eventingv1beta1.AddToScheme)
	AddToSchemes = append(AddToSchemes, eventingv1.AddToScheme)
	AddToSchemes = append(AddToSchemes, messagingv1.AddToScheme)
	AddToSchemes = append(AddToSchemes, sourcesv1.AddToScheme)
	AddToSchemes = append(AddToSchemes, sourcesv1beta2.AddToScheme)
}
