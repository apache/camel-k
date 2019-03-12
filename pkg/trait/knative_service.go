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
	"sort"
	"strconv"
	"strings"

	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/envvar"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	knativeServingClassAnnotation    = "autoscaling.knative.dev/class"
	knativeServingMetricAnnotation   = "autoscaling.knative.dev/metric"
	knativeServingTargetAnnotation   = "autoscaling.knative.dev/target"
	knativeServingMinScaleAnnotation = "autoscaling.knative.dev/minScale"
	knativeServingMaxScaleAnnotation = "autoscaling.knative.dev/maxScale"
)

type knativeServiceTrait struct {
	BaseTrait         `property:",squash"`
	Class             string `property:"autoscaling-class"`
	Metric            string `property:"autoscaling-metric"`
	Target            *int   `property:"autoscaling-target"`
	MinScale          *int   `property:"min-scale"`
	MaxScale          *int   `property:"max-scale"`
	Auto              *bool  `property:"auto"`
	ConfigurationType string `property:"configuration-type"`
	deployer          deployerTrait
}

func newKnativeServiceTrait() *knativeServiceTrait {
	return &knativeServiceTrait{
		BaseTrait: newBaseTrait("knative-service"),
	}
}

func (t *knativeServiceTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	strategy, err := e.DetermineControllerStrategy(t.ctx, t.client)
	if err != nil {
		return false, err
	}
	if strategy != ControllerStrategyKnativeService {
		return false, nil
	}

	deployment := e.Resources.GetDeployment(func(d *appsv1.Deployment) bool {
		if name, ok := d.ObjectMeta.Labels["camel.apache.org/integration"]; ok {
			return name == e.Integration.Name
		}
		return false
	})
	if deployment != nil {
		// A controller is already present for the integration
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		// Check the right value for minScale, as not all services are allowed to scale down to 0
		if t.MinScale == nil {
			sources, err := kubernetes.ResolveIntegrationSources(t.ctx, t.client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}

			meta := metadata.ExtractAll(e.CamelCatalog, sources)
			if !meta.RequiresHTTPService || !meta.PassiveEndpoints {
				single := 1
				t.MinScale = &single
			}
		}
	}

	dt := e.Catalog.GetTrait("deployer")
	if dt != nil {
		t.deployer = *dt.(*deployerTrait)
	}

	return true, nil
}

func (t *knativeServiceTrait) Apply(e *Environment) error {
	svc, err := t.getServiceFor(e)
	if err != nil {
		return err
	}

	maps := e.ComputeConfigMaps(t.deployer.ContainerImage)

	e.Resources.Add(svc)
	e.Resources.AddAll(maps)

	return nil
}

func (t *knativeServiceTrait) getServiceFor(e *Environment) (*serving.Service, error) {
	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	annotations := make(map[string]string)
	// Resolve registry host names when used
	annotations["alpha.image.policy.openshift.io/resolve-names"] = "*"

	//
	// Set Knative Scaling behavior
	//
	if t.Class != "" {
		annotations[knativeServingClassAnnotation] = t.Class
	}
	if t.Metric != "" {
		annotations[knativeServingMetricAnnotation] = t.Metric
	}
	if t.Target != nil {
		annotations[knativeServingTargetAnnotation] = strconv.Itoa(*t.Target)
	}
	if t.MinScale != nil {
		annotations[knativeServingMinScaleAnnotation] = strconv.Itoa(*t.MinScale)
	}
	if t.MaxScale != nil {
		annotations[knativeServingMaxScaleAnnotation] = strconv.Itoa(*t.MaxScale)
	}

	svc := serving.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: serving.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Integration.Name,
			Namespace:   e.Integration.Namespace,
			Labels:      labels,
			Annotations: e.Integration.Annotations,
		},
		Spec: serving.ServiceSpec{
			RunLatest: &serving.RunLatestType{
				Configuration: serving.ConfigurationSpec{
					RevisionTemplate: serving.RevisionTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      labels,
							Annotations: annotations,
						},
						Spec: serving.RevisionSpec{
							ServiceAccountName: e.Integration.Spec.ServiceAccountName,
							Container: corev1.Container{
								Image: e.Integration.Status.Image,
								Env:   make([]corev1.EnvVar, 0),
							},
						},
					},
				},
			},
		},
	}

	environment := &svc.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env

	// combine Environment of integration with context, integration
	// Environment has the priority
	VisitKeyValConfigurations("env", e.IntegrationContext, e.Integration, func(key string, value string) {
		envvar.SetVal(environment, key, value)
	})

	// set env vars needed by the runtime
	envvar.SetVal(environment, "JAVA_MAIN_CLASS", "org.apache.camel.k.jvm.Application")

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	envvar.SetVal(environment, "CAMEL_K_DIGEST", e.Integration.Status.Digest)

	// optimizations
	envvar.SetVal(environment, "AB_JOLOKIA_OFF", True)

	if t.ConfigurationType == "volume" {
		t.bindToVolumes(e, &svc)
	} else if err := t.bindToEnvVar(e, &svc); err != nil {
		return nil, err
	}

	// add env vars from traits
	for _, envVar := range e.EnvVars {
		envvar.SetVar(&svc.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Env, envVar)
	}

	// Add mounted volumes as resources
	for _, m := range svc.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.VolumeMounts {
		e.Classpath.Add(m.MountPath)
	}

	cp := e.Classpath.List()

	// keep classpath sorted
	sort.Strings(cp)

	// set the classpath
	envvar.SetVal(environment, "JAVA_CLASSPATH", strings.Join(cp, ":"))

	return &svc, nil
}
