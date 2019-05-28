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
	"github.com/apache/camel-k/pkg/util/envvar"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type deploymentTrait struct {
	BaseTrait `property:",squash"`
	deployer  deployerTrait
}

func newDeploymentTrait() *deploymentTrait {
	return &deploymentTrait{
		BaseTrait: newBaseTrait("deployment"),
	}
}

func (t *deploymentTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	enabled := false

	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		//
		// Don't deploy when a different strategy is needed (e.g. Knative)
		//
		strategy, err := e.DetermineControllerStrategy(t.ctx, t.client)
		if err != nil {
			return false, err
		}

		enabled = strategy == ControllerStrategyDeployment
	} else if e.IntegrationContextInPhase(v1alpha1.IntegrationContextPhaseReady) &&
		e.IntegrationInPhase(v1alpha1.IntegrationPhaseBuildingContext, v1alpha1.IntegrationPhaseResolvingContext) {
		enabled = true
	}

	if enabled {
		dt := e.Catalog.GetTrait("deployer")
		if dt != nil {
			t.deployer = *dt.(*deployerTrait)
		}
	}

	return enabled, nil
}

func (t *deploymentTrait) Apply(e *Environment) error {
	if e.IntegrationContextInPhase(v1alpha1.IntegrationContextPhaseReady) &&
		e.IntegrationInPhase(v1alpha1.IntegrationPhaseBuildingContext, v1alpha1.IntegrationPhaseResolvingContext) {

		e.PostProcessors = append(e.PostProcessors, func(environment *Environment) error {
			// trigger integration deploy
			e.Integration.Status.Phase = v1alpha1.IntegrationPhaseDeploying
			return nil
		})

		return nil
	}

	if e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseDeploying) {
		e.Resources.AddAll(e.ComputeConfigMaps())
		e.Resources.Add(t.getDeploymentFor(e))
	}

	return nil
}

// **********************************
//
// Deployment
//
// **********************************

func (t *deploymentTrait) getDeploymentFor(e *Environment) *appsv1.Deployment {
	paths := e.ComputeSourcesURI()
	environment := make([]corev1.EnvVar, 0)

	// combine Environment of integration with platform, context, integration
	for key, value := range e.CollectConfigurationPairs("env") {
		envvar.SetVal(&environment, key, value)
	}

	// set env vars needed by the runtime
	envvar.SetVal(&environment, "JAVA_MAIN_CLASS", "org.apache.camel.k.jvm.Application")

	// camel-k runtime
	envvar.SetVal(&environment, "CAMEL_K_ROUTES", strings.Join(paths, ","))
	envvar.SetVal(&environment, "CAMEL_K_CONF", "/etc/camel/conf/application.properties")
	envvar.SetVal(&environment, "CAMEL_K_CONF_D", "/etc/camel/conf.d")

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	envvar.SetVal(&environment, "CAMEL_K_DIGEST", e.Integration.Status.Digest)

	// add env vars from traits
	for _, envVar := range e.EnvVars {
		envvar.SetVar(&environment, envVar)
	}

	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	// create a copy to avoid sharing the underlying annotation map
	annotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for k, v := range FilterTransferableAnnotations(e.Integration.Annotations) {
			annotations[k] = v
		}
	}

	// Resolve registry host names when used
	annotations["alpha.image.policy.openshift.io/resolve-names"] = "*"

	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Integration.Name,
			Namespace:   e.Integration.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: e.Integration.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: e.Integration.Spec.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  e.Integration.Name,
							Image: e.Integration.Status.Image,
							Env:   environment,
						},
					},
				},
			},
		},
	}

	e.ConfigureVolumesAndMounts(
		&deployment.Spec.Template.Spec.Volumes,
		&deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
	)

	return &deployment
}
