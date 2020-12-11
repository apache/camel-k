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
	"fmt"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/ghodss/yaml"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/strategicpatch"

	//"github.com/ghodss/yaml"
	//	cmd "github.com/apache/camel-k/pkg/cmd"

	appsv1 "k8s.io/api/apps/v1"
)

// The Pod trait allows to modify the custom PodTemplateSpec resource for the Integration pods
//
// +camel-k:trait=pdb
type podTrait struct {
	BaseTrait `property:",squash"`
	Template  string `property:"template"`
}

func newPodTrait() Trait {
	return &podTrait{
		BaseTrait: NewBaseTrait("pod", 1800),
	}
}

func (t *podTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		return false, nil
	}

	if t.Template == "" {
		return false, fmt.Errorf("template must be specified")
	}

	if _, err := t.parseTemplate(); err != nil {
		return false, err
	}

	return e.IntegrationInPhase(
		v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning,
	), nil
}

func (t *podTrait) Apply(e *Environment) error {
	var deployment *appsv1.Deployment

	e.Resources.VisitDeployment(func(d *appsv1.Deployment) {
		if d.Name == e.Integration.Name {
			deployment = d
		}
	})

	modifiedTemplate, err := t.mergeIntoTemplateSpec(deployment.Spec.Template, []byte(t.Template))
	if err != nil {
		return err
	}
	deployment.Spec.Template = *modifiedTemplate
	return nil
}

func (t *podTrait) parseTemplate() (*v12.PodTemplateSpec, error) {
	var template *v12.PodTemplateSpec

	if err := yaml.Unmarshal([]byte(t.Template), &template); err != nil {
		return nil, err
	}
	return template, nil
}

func (t *podTrait) mergeIntoTemplateSpec(template v12.PodTemplateSpec, changesBytes []byte) (mergedTemplate *v12.PodTemplateSpec, err error) {
	patch, err := yaml.YAMLToJSON(changesBytes)
	if err != nil {
		return nil, err
	}

	sourceJson, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}

	patched, err := strategicpatch.StrategicMergePatch(sourceJson, patch, v12.PodTemplateSpec{})
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(patched, &mergedTemplate)
	return
}
