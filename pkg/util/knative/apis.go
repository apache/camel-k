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

package knative

import (
	knativev1 "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	"github.com/apache/camel-k/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// KnownChannelKinds are known channel kinds belonging to Knative.
	KnownChannelKinds = []GroupVersionKindResource{
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Channel",
				Group:   "messaging.knative.dev",
				Version: "v1",
			},
			Resource: "channels",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Channel",
				Group:   "messaging.knative.dev",
				Version: "v1beta1",
			},
			Resource: "channels",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "InMemoryChannel",
				Group:   "messaging.knative.dev",
				Version: "v1",
			},
			Resource: "inmemorychannels",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "InMemoryChannel",
				Group:   "messaging.knative.dev",
				Version: "v1beta1",
			},
			Resource: "inmemorychannels",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "KafkaChannel",
				Group:   "messaging.knative.dev",
				Version: "v1beta1",
			},
			Resource: "kafkachannels",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "KafkaChannel",
				Group:   "messaging.knative.dev",
				Version: "v1alpha1",
			},
			Resource: "kafkachannels",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "NatssChannel",
				Group:   "messaging.knative.dev",
				Version: "v1alpha1",
			},
			Resource: "natsschannels",
		},
	}

	// KnownEndpointKinds are known endpoint kinds belonging to Knative.
	KnownEndpointKinds = []GroupVersionKindResource{
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Service",
				Group:   "serving.knative.dev",
				Version: "v1",
			},
			Resource: "services",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Service",
				Group:   "serving.knative.dev",
				Version: "v1beta1",
			},
			Resource: "services",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Service",
				Group:   "serving.knative.dev",
				Version: "v1alpha1",
			},
			Resource: "services",
		},
	}

	// KnownBrokerKinds are known broker kinds belonging to Knative.
	KnownBrokerKinds = []GroupVersionKindResource{
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Broker",
				Group:   "eventing.knative.dev",
				Version: "v1",
			},
			Resource: "brokers",
		},
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Broker",
				Group:   "eventing.knative.dev",
				Version: "v1beta1",
			},
			Resource: "brokers",
		},
	}

	// RequiredKinds are Knative kinds used by Camel K for materializing integrations.
	// They must be present on the cluster.
	RequiredKinds = []GroupVersionKindResource{
		{
			GroupVersionKind: schema.GroupVersionKind{
				Kind:    "Service",
				Group:   "serving.knative.dev",
				Version: "v1",
			},
			Resource: "services",
		},
	}
)

// GroupVersionKindResource --.
type GroupVersionKindResource struct {
	schema.GroupVersionKind
	Resource string
}

func init() {
	// Channels are also endpoints
	KnownEndpointKinds = append(KnownEndpointKinds, KnownChannelKinds...)
	// Let's add the brokers as last
	KnownEndpointKinds = append(KnownEndpointKinds, KnownBrokerKinds...)
}

// FillMissingReferenceData returns all possible combinations of ObjectReference that can be obtained by filling
// the missing fields with known data.
func FillMissingReferenceData(serviceType knativev1.CamelServiceType,
	ref v1.ObjectReference) []v1.ObjectReference {
	var refs []v1.ObjectReference
	switch serviceType {
	case knativev1.CamelServiceTypeChannel:
		refs = fillMissingReferenceDataWith(KnownChannelKinds, ref)
	case knativev1.CamelServiceTypeEndpoint:
		refs = fillMissingReferenceDataWith(KnownEndpointKinds, ref)
	case knativev1.CamelServiceTypeEvent:
		refs = fillMissingReferenceDataWith(KnownBrokerKinds, ref)
	}

	return refs
}

func GetServiceType(ref v1.ObjectReference) (*knativev1.CamelServiceType, error) {
	refGV, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, err
	}

	for _, c := range KnownChannelKinds {
		if c.Group == refGV.Group && c.Kind == ref.Kind {
			channelType := knativev1.CamelServiceTypeChannel
			return &channelType, nil
		}
	}

	for _, c := range KnownBrokerKinds {
		if c.Group == refGV.Group && c.Kind == ref.Kind {
			eventType := knativev1.CamelServiceTypeEvent
			return &eventType, nil
		}
	}

	for _, c := range KnownEndpointKinds {
		if c.Group == refGV.Group && c.Kind == ref.Kind {
			endpointType := knativev1.CamelServiceTypeEndpoint
			return &endpointType, nil
		}
	}

	return nil, nil
}

func fillMissingReferenceDataWith(serviceTypes []GroupVersionKindResource,
	ref v1.ObjectReference) []v1.ObjectReference {
	list := make([]v1.ObjectReference, 0)
	switch {
	case ref.APIVersion == "" && ref.Kind == "":
		for _, st := range serviceTypes {
			refCopy := ref.DeepCopy()
			refCopy.APIVersion = st.GroupVersion().String()
			refCopy.Kind = st.Kind
			list = append(list, *refCopy)
		}
	case ref.APIVersion == "":
		for _, gv := range getGroupVersions(serviceTypes, ref.Kind) {
			refCopy := ref.DeepCopy()
			refCopy.APIVersion = gv
			list = append(list, *refCopy)
		}
	case ref.Kind == "":
		for _, k := range getKinds(serviceTypes, ref.APIVersion) {
			refCopy := ref.DeepCopy()
			refCopy.Kind = k
			list = append(list, *refCopy)
		}
	default:
		list = append(list, ref)
	}
	return list
}

func getGroupVersions(serviceTypes []GroupVersionKindResource, kind string) []string {
	res := make([]string, 0)
	for _, st := range serviceTypes {
		if st.Kind == kind {
			util.StringSliceUniqueAdd(&res, st.GroupVersion().String())
		}
	}
	return res
}

func getKinds(serviceTypes []GroupVersionKindResource, apiVersion string) []string {
	res := make([]string, 0)
	for _, st := range serviceTypes {
		if st.GroupVersion().String() == apiVersion {
			util.StringSliceUniqueAdd(&res, st.Kind)
		}
	}
	return res
}
