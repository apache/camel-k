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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

type environmentTrait struct {
	BaseTrait     `property:",squash"`
	ContainerMeta bool `property:"container-meta"`
}

const (
	envVarNamespace = "NAMESPACE"
	envVarPodName   = "POD_NAME"
)

func newEnvironmentTrait() *environmentTrait {
	return &environmentTrait{
		BaseTrait: BaseTrait{
			id: ID("environment"),
		},
		ContainerMeta: true,
	}
}

func (t *environmentTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || *t.Enabled {
		return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
	}

	return false, nil
}

func (t *environmentTrait) Apply(e *Environment) error {
	if t.ContainerMeta {
		e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
			for i := 0; i < len(deployment.Spec.Template.Spec.Containers); i++ {
				c := &deployment.Spec.Template.Spec.Containers[i]
				c.Env = append(c.Env, v1.EnvVar{
					Name: envVarNamespace,
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				})
				c.Env = append(c.Env, v1.EnvVar{
					Name: envVarPodName,
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				})
			}
		})
	}

	return nil
}
