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
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// labelsTrait transfers labels on the integration onto the owned resources being created
type labelsTrait struct {
	BaseTrait `property:",squash"`

	TargetLabels string `property:"target-labels"`
}

func newLabelsTrait() *labelsTrait {
	return &labelsTrait{
		BaseTrait: newBaseTrait("labels"),
	}
}

func (t *labelsTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *labelsTrait) Apply(e *Environment) error {
	targetLabels := map[string]string{}
	for _, k := range strings.Split(t.TargetLabels, ",") {
		if v, ok := e.Integration.Labels[k]; ok {
			targetLabels[k] = v
		}
	}
	if len(targetLabels) == 0 {
		return nil
	}
	e.Resources.VisitMetaObject(func(res metav1.Object) {
		labels := res.GetLabels()
		for k, v := range targetLabels {
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}
		res.SetLabels(labels)
	})
	return nil
}
