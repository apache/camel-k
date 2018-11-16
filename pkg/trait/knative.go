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
	"github.com/apache/camel-k/pkg/metadata"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	eventing "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
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

func (t *knativeTrait) autoconfigure(environment *environment, resources *kubernetes.Collection) error {
	if t.Sources == "" {
		channels := t.getSourceChannels(environment)
		t.Sources = strings.Join(channels, ",")
	}
	return nil
}

func (t *knativeTrait) beforeDeploy(environment *environment, resources *kubernetes.Collection) error {
	for _, sub := range t.getSubscriptionsFor(environment) {
		resources.Add(sub)
	}
	resources.Add(t.getServiceFor(environment))
	return nil
}

func (t *knativeTrait) getServiceFor(e *environment) *serving.Service {
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

func (t *knativeTrait) getSubscriptionsFor(e *environment) []*eventing.Subscription {
	channels := t.getConfiguredSourceChannels()
	subs := make([]*eventing.Subscription, 0)
	for _, ch := range channels {
		subs = append(subs, t.getSubscriptionFor(e, ch))
	}
	return subs
}

func (*knativeTrait) getSubscriptionFor(e *environment, channel string) *eventing.Subscription {
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

func (t *knativeTrait) getConfigurationSerialized(e *environment) string {
	env := t.getConfiguration(e)
	res, err := json.Marshal(env)
	if err != nil {
		logrus.Warning("Unable to serialize Knative configuration", err)
		return ""
	}
	return string(res)
}

func (t *knativeTrait) getConfiguration(e *environment) knativeutil.CamelEnvironment {
	sourceChannels := t.getConfiguredSourceChannels()
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

func (t *knativeTrait) getConfiguredSourceChannels() []string {
	channels := make([]string, 0)
	for _, ch := range strings.Split(t.Sources, ",") {
		cht := strings.Trim(ch, " \t\"")
		if cht != "" {
			channels = append(channels, cht)
		}
	}
	return channels
}

func (*knativeTrait) getSourceChannels(e *environment) []string {
	meta := metadata.Extract(e.Integration.Spec.Source)
	return knativeutil.ExtractChannelNames(meta.FromURIs)
}
