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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

	"github.com/apache/camel-k/pkg/metadata"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
	eventing "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type knativeTrait struct {
	BaseTrait `property:",squash"`
	Sources   string `property:"sources"`
}

func newKnativeTrait() *knativeTrait {
	return &knativeTrait{
		BaseTrait: newBaseTrait("knative"),
	}
}

func (t *knativeTrait) appliesTo(e *Environment) bool {
	return e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (t *knativeTrait) autoconfigure(e *Environment) error {
	if t.Sources == "" {
		channels := getSourceChannels(e)
		t.Sources = strings.Join(channels, ",")
	}
	return nil
}

func (t *knativeTrait) apply(e *Environment) error {
	for _, sub := range t.getSubscriptionsFor(e) {
		e.Resources.Add(sub)
	}
	e.Resources.Add(t.getServiceFor(e))
	return nil
}

func (t *knativeTrait) getServiceFor(e *Environment) *serving.Service {
	// combine properties of integration with context, integration
	// properties have the priority
	properties := CombineConfigurationAsMap("property", e.Context, e.Integration)

	// combine Environment of integration with context, integration
	// Environment has the priority
	environment := CombineConfigurationAsMap("env", e.Context, e.Integration)

	sources := make([]string, 0, len(e.Integration.Spec.Sources))
	for i, s := range e.Integration.Spec.Sources {
		envName := fmt.Sprintf("KAMEL_K_ROUTE_%03d", i)
		environment[envName] = s.Content

		src := fmt.Sprintf("env:%s", envName)
		if s.Language != "" {
			src = src + "?language=" + string(s.Language)
		}

		sources = append(sources, src)
	}

	// set env vars needed by the runtime
	environment["JAVA_MAIN_CLASS"] = "org.apache.camel.k.jvm.Application"

	// camel-k runtime
	environment["CAMEL_K_ROUTES"] = strings.Join(sources, ",")
	environment["CAMEL_K_CONF"] = "env:CAMEL_K_PROPERTIES"
	environment["CAMEL_K_PROPERTIES"] = PropertiesString(properties)

	// add a dummy env var to trigger deployment if everything but the code
	// has been changed
	environment["CAMEL_K_DIGEST"] = e.Integration.Status.Digest

	// optimizations
	environment["AB_JOLOKIA_OFF"] = "true"

	// Knative integration
	environment["CAMEL_KNATIVE_CONFIGURATION"] = t.getConfigurationSerialized(e)

	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
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
						Spec: serving.RevisionSpec{
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

func (t *knativeTrait) getSubscriptionsFor(e *Environment) []*eventing.Subscription {
	channels := getConfiguredSourceChannels(t.Sources)
	subs := make([]*eventing.Subscription, 0)
	for _, ch := range channels {
		subs = append(subs, t.getSubscriptionFor(e, ch))
	}
	return subs
}

func (*knativeTrait) getSubscriptionFor(e *Environment, channel string) *eventing.Subscription {
	return &eventing.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventing.SchemeGroupVersion.String(),
			Kind:       "Subscription",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: e.Integration.Namespace,
			Name:      channel + "-" + e.Integration.Name,
		},
		Spec: eventing.SubscriptionSpec{
			Channel: corev1.ObjectReference{
				APIVersion: eventing.SchemeGroupVersion.String(),
				Kind:       "Channel",
				Name:       channel,
			},
			Subscriber: &eventing.SubscriberSpec{
				Ref: &corev1.ObjectReference{
					APIVersion: serving.SchemeGroupVersion.String(),
					Kind:       "Service",
					Name:       e.Integration.Name,
				},
			},
		},
	}
}

func (t *knativeTrait) getConfigurationSerialized(e *Environment) string {
	env := t.getConfiguration(e)
	res, err := json.Marshal(env)
	if err != nil {
		logrus.Warning("Unable to serialize Knative configuration", err)
		return ""
	}
	return string(res)
}

func (t *knativeTrait) getConfiguration(e *Environment) knativeutil.CamelEnvironment {
	sourceChannels := getConfiguredSourceChannels(t.Sources)
	env := knativeutil.NewCamelEnvironment()
	for _, ch := range sourceChannels {
		svc := knativeutil.CamelServiceDefinition{
			Name:        ch,
			Host:        "0.0.0.0",
			Port:        8080,
			Protocol:    knativeutil.CamelProtocolHTTP,
			ServiceType: knativeutil.CamelServiceTypeChannel,
			Metadata: map[string]string{
				knativeutil.CamelMetaServicePath: "/",
			},
		}
		env.Services = append(env.Services, svc)
	}
	// Adding default endpoint
	defSvc := knativeutil.CamelServiceDefinition{
		Name:        "default",
		Host:        "0.0.0.0",
		Port:        8080,
		Protocol:    knativeutil.CamelProtocolHTTP,
		ServiceType: knativeutil.CamelServiceTypeEndpoint,
		Metadata: map[string]string{
			knativeutil.CamelMetaServicePath: "/",
		},
	}
	env.Services = append(env.Services, defSvc)
	return env
}

func getConfiguredSourceChannels(sources string) []string {
	channels := make([]string, 0)
	for _, ch := range strings.Split(sources, ",") {
		cht := strings.Trim(ch, " \t\"")
		if cht != "" {
			channels = append(channels, cht)
		}
	}
	return channels
}

func getSourceChannels(e *Environment) []string {
	channels := make([]string, 0)

	metadata.Each(e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
		channels = append(channels, knativeutil.ExtractChannelNames(meta.FromURIs)...)
		return true
	})

	return channels
}
