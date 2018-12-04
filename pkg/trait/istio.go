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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

type istioTrait struct {
	BaseTrait `property:",squash"`
	Allow     string `property:"allow"`
}

const (
	istioIncludeAnnotation = "traffic.sidecar.istio.io/includeOutboundIPRanges"
)

func newIstioTrait() *istioTrait {
	return &istioTrait{
		BaseTrait: newBaseTrait("istio"),
		Allow:     "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16",
	}
}

func (t *istioTrait) appliesTo(e *Environment) bool {
	return e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (t *istioTrait) apply(e *Environment) error {
	if t.Allow != "" {
		e.Resources.VisitDeployment(func(d *appsv1.Deployment) {
			d.Spec.Template.Annotations = t.injectIstioAnnotation(d.Spec.Template.Annotations)
		})
		e.Resources.VisitKnativeConfigurationSpec(func(cs *serving.ConfigurationSpec) {
			cs.RevisionTemplate.Annotations = t.injectIstioAnnotation(cs.RevisionTemplate.Annotations)
		})
	}
	return nil
}

func (t *istioTrait) injectIstioAnnotation(annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[istioIncludeAnnotation] = t.Allow
	return annotations
}
