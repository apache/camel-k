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
	"k8s.io/utils/pointer"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
)

type istioTrait struct {
	BaseTrait
	traitv1.IstioTrait `property:",squash"`
}

const (
	istioSidecarInjectAnnotation    = "sidecar.istio.io/inject"
	istioOutboundIPRangesAnnotation = "traffic.sidecar.istio.io/includeOutboundIPRanges"
)

func newIstioTrait() Trait {
	return &istioTrait{
		BaseTrait: NewBaseTrait("istio", 2300),
		IstioTrait: traitv1.IstioTrait{
			Allow: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
		},
	}
}

func (t *istioTrait) Configure(e *Environment) (bool, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, false) {
		return false, nil
	}

	return e.IntegrationInRunningPhases(), nil
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
