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
	"github.com/apache/camel-k/pkg/util/kubernetes"
	knative "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type knativeTrait struct {
	BaseTrait `property:",squash"`
}

func newKnativeTrait() *knativeTrait {
	return &knativeTrait{
		BaseTrait: newBaseTrait("knative"),
	}
}

func (t *knativeTrait) autoconfigure(environment *environment, resources *kubernetes.Collection) error {
	if t.Enabled == nil {
		// disable by default
		status := false
		t.Enabled = &status
	}
	return nil
}

func (t *knativeTrait) beforeDeploy(environment *environment, resources *kubernetes.Collection) error {
	resources.Add(t.getServiceFor(environment))
	return nil
}

func (*knativeTrait) getServiceFor(e *environment) *knative.Service {
	// combine properties of integration with context, integration
	// properties have the priority
	properties := CombineConfigurationAsMap("property", e.Context, e.Integration)

	// combine environment of integration with context, integration
	// environment has the priority
	environment := CombineConfigurationAsMap("env", e.Context, e.Integration)

	// set env vars needed by the runtime
	environment["JAVA_MAIN_CLASS"] = "org.apache.camel.k.jvm.Application"

	// camel-k runtime
	environment["CAMEL_K_ROUTES_URI"] = "inline:" + e.Integration.Spec.Source.Content
	environment["CAMEL_K_ROUTES_LANGUAGE"] = string(e.Integration.Spec.Source.Language)
	environment["CAMEL_K_CONF"] = "inline:" + PropertiesString(properties)
	environment["CAMEL_K_CONF_D"] = "/etc/camel/conf.d"

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	environment["CAMEL_K_DIGEST"] = e.Integration.Status.Digest

	// optimizations
	environment["AB_JOLOKIA_OFF"] = "true"

	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	svc := knative.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: knative.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Integration.Name,
			Namespace:   e.Integration.Namespace,
			Labels:      labels,
			Annotations: e.Integration.Annotations,
		},
		Spec: knative.ServiceSpec{
			RunLatest: &knative.RunLatestType{
				Configuration: knative.ConfigurationSpec{
					RevisionTemplate: knative.RevisionTemplateSpec{
						Spec: knative.RevisionSpec{
							Container: corev1.Container{
								Image: e.Integration.Status.Image,
								Env:   EnvironmentAsEnvVarSlice(environment),
							},
						},
					},
				},
			},
		},
	}

	return &svc
}
