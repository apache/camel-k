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

	"github.com/apache/camel-k/v2/pkg/util/boolean"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

const (
	istioTraitID    = "istio"
	istioTraitOrder = 2300
)

type istioTrait struct {
	BaseTrait
	traitv1.IstioTrait `property:",squash"`
}

const (
	istioSidecarInjectAnnotation    = "sidecar.istio.io/inject"
	istioOutboundIPRangesAnnotation = "traffic.sidecar.istio.io/includeOutboundIPRanges"

	defaultAllow = "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
)

func newIstioTrait() Trait {
	return &istioTrait{
		BaseTrait: NewBaseTrait(istioTraitID, istioTraitOrder),
	}
}

func (t *istioTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) {
		return false, nil, nil
	}

	return e.IntegrationInRunningPhases(), nil, nil
}

func (t *istioTrait) Apply(e *Environment) error {
	if t.getAllow() != "" {
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
	annotations[istioOutboundIPRangesAnnotation] = t.getAllow()
	if includeInject {
		annotations[istioSidecarInjectAnnotation] = boolean.TrueString
	}
	if t.Inject != nil {
		annotations[istioSidecarInjectAnnotation] = strconv.FormatBool(*t.Inject)
	}
	return annotations
}

func (t *istioTrait) getAllow() string {
	if t.Allow == "" {
		return defaultAllow
	}

	return t.Allow
}
