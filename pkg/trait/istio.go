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

package trait

import (
	"strconv"

	appsv1 "k8s.io/api/apps/v1"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// The Istio trait allows configuring properties related to the Istio service mesh,
// such as sidecar injection and outbound IP ranges.
//
// +camel-k:trait=istio.
type istioTrait struct {
	BaseTrait `property:",squash"`
	// Configures a (comma-separated) list of CIDR subnets that should not be intercepted by the Istio proxy (`10.0.0.0/8,172.16.0.0/12,192.168.0.0/16` by default).
	Allow string `property:"allow" json:"allow,omitempty"`
	// Forces the value for labels `sidecar.istio.io/inject`. By default the label is set to `true` on deployment and not set on Knative Service.
	Inject *bool `property:"inject" json:"inject,omitempty"`
}

const (
	istioSidecarInjectAnnotation    = "sidecar.istio.io/inject"
	istioOutboundIPRangesAnnotation = "traffic.sidecar.istio.io/includeOutboundIPRanges"
)

func newIstioTrait() Trait {
	return &istioTrait{
		BaseTrait: NewBaseTrait("istio", 2300),
		Allow:     "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
	}
}

func (t *istioTrait) Configure(e *Environment) (bool, error) {
	if IsTrue(t.Enabled) {
		return e.IntegrationInRunningPhases(), nil
	}

	return false, nil
}

func (t *istioTrait) Apply(e *Environment) error {
	if t.Allow != "" {
		e.Resources.VisitDeployment(func(d *appsv1.Deployment) {
			d.Spec.Template.Annotations = t.injectIstioAnnotation(d.Spec.Template.Annotations, true)
		})
		e.Resources.VisitKnativeConfigurationSpec(func(cs *servingv1.ConfigurationSpec) {
			cs.Template.Annotations = t.injectIstioAnnotation(cs.Template.Annotations, false)
		})
	}
	return nil
}

func (t *istioTrait) injectIstioAnnotation(annotations map[string]string, includeInject bool) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[istioOutboundIPRangesAnnotation] = t.Allow
	if includeInject {
		annotations[istioSidecarInjectAnnotation] = True
	}
	if t.Inject != nil {
		annotations[istioSidecarInjectAnnotation] = strconv.FormatBool(*t.Inject)
	}
	return annotations
}
