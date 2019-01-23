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
	"strconv"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/envvar"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	knativeMinScaleAnnotation = "autoscaling.knative.dev/minScale"
	knativeMaxScaleAnnotation = "autoscaling.knative.dev/maxScale"
)

type knativeServiceTrait struct {
	BaseTrait `property:",squash"`
	MinScale  *int  `property:"min-scale"`
	MaxScale  *int  `property:"max-scale"`
	Auto      *bool `property:"auto"`
}

func newKnativeServiceTrait() *knativeServiceTrait {
	return &knativeServiceTrait{
		BaseTrait: BaseTrait{
			id: ID("knative-service"),
		},
	}
}

func (t *knativeServiceTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	var strategy ControllerStrategy
	var err error
	if strategy, err = e.DetermineControllerStrategy(t.ctx, t.client); err != nil {
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
			var sources []v1alpha1.SourceSpec
			if sources, err = e.ResolveSources(t.ctx, t.client); err != nil {
				return false, err
			}
			meta := metadata.ExtractAll(sources)

			if !meta.RequiresHTTPService || !meta.PassiveEndpoints {
				single := 1
				t.MinScale = &single
			}
		}
	}

	return true, nil
}

func (t *knativeServiceTrait) Apply(e *Environment) error {
	svc, err := t.getServiceFor(e)
	if err != nil {
		return err
	}

	e.Resources.Add(svc)

	return nil
}

func (t *knativeServiceTrait) getServiceFor(e *Environment) (*serving.Service, error) {
	// combine properties of integration with context, integration
	// properties have the priority
	properties := ""

	VisitKeyValConfigurations("property", e.Context, e.Integration, func(key string, val string) {
		properties += fmt.Sprintf("%s=%s\n", key, val)
	})

	environment := make([]corev1.EnvVar, 0)

	// combine Environment of integration with context, integration
	// Environment has the priority
	VisitKeyValConfigurations("env", e.Context, e.Integration, func(key string, value string) {
		envvar.SetVal(&environment, key, value)
	})

	sourcesSpecs, err := e.ResolveSources(t.ctx, t.client)
	if err != nil {
		return nil, err
	}

	sources := make([]string, 0, len(e.Integration.Spec.Sources))
	for i, s := range sourcesSpecs {
		if s.Content == "" {
			t.L.Debug("Source %s has and empty content", s.Name)
		}

		envName := fmt.Sprintf("CAMEL_K_ROUTE_%03d", i)
		envvar.SetVal(&environment, envName, s.Content)

		params := make([]string, 0)
		if s.InferLanguage() != "" {
			params = append(params, "language="+string(s.InferLanguage()))
		}
		if s.Compression {
			params = append(params, "compression=true")
		}

		src := fmt.Sprintf("env:%s", envName)
		if len(params) > 0 {
			src = fmt.Sprintf("%s?%s", src, strings.Join(params, "&"))
		}

		sources = append(sources, src)
	}

	for i, r := range e.Integration.Spec.Resources {
		if r.Type != v1alpha1.ResourceTypeData {
			continue
		}

		envName := fmt.Sprintf("CAMEL_K_RESOURCE_%03d", i)
		envvar.SetVal(&environment, envName, r.Content)

		params := make([]string, 0)
		if r.Compression {
			params = append(params, "compression=true")
		}

		envValue := fmt.Sprintf("env:%s", envName)
		if len(params) > 0 {
			envValue = fmt.Sprintf("%s?%s", envValue, strings.Join(params, "&"))
		}

		envName = r.Name
		envName = strings.ToUpper(envName)
		envName = strings.Replace(envName, "-", "_", -1)
		envName = strings.Replace(envName, ".", "_", -1)
		envName = strings.Replace(envName, " ", "_", -1)

		envvar.SetVal(&environment, envName, envValue)
	}

	// set env vars needed by the runtime
	envvar.SetVal(&environment, "JAVA_MAIN_CLASS", "org.apache.camel.k.jvm.Application")

	// camel-k runtime
	envvar.SetVal(&environment, "CAMEL_K_ROUTES", strings.Join(sources, ","))
	envvar.SetVal(&environment, "CAMEL_K_CONF", "env:CAMEL_K_PROPERTIES")
	envvar.SetVal(&environment, "CAMEL_K_PROPERTIES", properties)

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	envvar.SetVal(&environment, "CAMEL_K_DIGEST", e.Integration.Status.Digest)

	// optimizations
	envvar.SetVal(&environment, "AB_JOLOKIA_OFF", True)

	// add env vars from traits
	for _, envVar := range e.EnvVars {
		envvar.SetVar(&environment, envVar)
	}

	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	annotations := make(map[string]string)
	// Resolve registry host names when used
	annotations["alpha.image.policy.openshift.io/resolve-names"] = "*"
	if t.MinScale != nil {
		annotations[knativeMinScaleAnnotation] = strconv.Itoa(*t.MinScale)
	}
	if t.MaxScale != nil {
		annotations[knativeMaxScaleAnnotation] = strconv.Itoa(*t.MaxScale)
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
								Env:   environment,
							},
						},
					},
				},
			},
		},
	}

	return &svc, nil
}
