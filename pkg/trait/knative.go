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

	"github.com/apache/camel-k/pkg/util/envvar"

	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

	knativeapi "github.com/apache/camel-k/pkg/apis/camel/v1alpha1/knative"
	"github.com/apache/camel-k/pkg/metadata"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
	eventing "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	knativeMinScaleAnnotation = "autoscaling.knative.dev/minScale"
	knativeMaxScaleAnnotation = "autoscaling.knative.dev/maxScale"
)

type knativeTrait struct {
	BaseTrait     `property:",squash"`
	Configuration string `property:"configuration"`
	Sources       string `property:"sources"`
	Sinks         string `property:"sinks"`
	MinScale      *int   `property:"minScale"`
	MaxScale      *int   `property:"maxScale"`
	Auto          *bool  `property:"auto"`
}

func newKnativeTrait() *knativeTrait {
	return &knativeTrait{
		BaseTrait: BaseTrait{
			id: ID("knative"),
		},
	}
}

func (t *knativeTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if t.Sources == "" {
			channels := t.getSourceChannels(e)
			t.Sources = strings.Join(channels, ",")
		}
		if t.Sinks == "" {
			channels := t.getSinkChannels(e)
			t.Sinks = strings.Join(channels, ",")
		}
		// Check the right value for minScale, as not all services are allowed to scale down to 0
		if t.MinScale == nil {
			meta := metadata.ExtractAll(e.Integration.Spec.Sources)
			if !meta.RequiresHTTPService || !meta.PassiveEndpoints {
				single := 1
				t.MinScale = &single
			}
		}
	}

	return true, nil
}

func (t *knativeTrait) Apply(e *Environment) error {
	if err := t.prepareEnvVars(e); err != nil {
		return err
	}
	for _, sub := range t.getSubscriptionsFor(e) {
		e.Resources.Add(sub)
	}

	svc := t.getServiceFor(e)
	e.Resources.Add(svc)

	return nil
}

func (t *knativeTrait) prepareEnvVars(e *Environment) error {
	// common env var for Knative integration
	conf, err := t.getConfigurationSerialized(e)
	if err != nil {
		return err
	}

	envvar.SetVal(&e.EnvVars, "CAMEL_KNATIVE_CONFIGURATION", conf)

	return nil
}

func (t *knativeTrait) getServiceFor(e *Environment) *serving.Service {
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

	sources := make([]string, 0, len(e.Integration.Spec.Sources))
	for i, s := range e.Integration.Spec.Sources {
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

	return &svc
}

func (t *knativeTrait) getSubscriptionsFor(e *Environment) []*eventing.Subscription {
	channels := t.getConfiguredSourceChannels()
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

func (t *knativeTrait) getConfigurationSerialized(e *Environment) (string, error) {
	env, err := t.getConfiguration(e)
	if err != nil {
		return "", errors.Wrap(err, "unable fetch environment configuration")
	}
	return env.Serialize()
}

func (t *knativeTrait) getConfiguration(e *Environment) (knativeapi.CamelEnvironment, error) {
	env := knativeapi.NewCamelEnvironment()
	if t.Configuration != "" {
		if err := env.Deserialize(t.Configuration); err != nil {
			return knativeapi.CamelEnvironment{}, err
		}
	}

	// Sources
	sourceChannels := t.getConfiguredSourceChannels()
	for _, ch := range sourceChannels {
		if env.ContainsService(ch, knativeapi.CamelServiceTypeChannel) {
			continue
		}
		svc := knativeapi.CamelServiceDefinition{
			Name:        ch,
			Host:        "0.0.0.0",
			Port:        8080,
			Protocol:    knativeapi.CamelProtocolHTTP,
			ServiceType: knativeapi.CamelServiceTypeChannel,
			Metadata: map[string]string{
				knativeapi.CamelMetaServicePath: "/",
			},
		}
		env.Services = append(env.Services, svc)
	}
	// Sinks
	sinkChannels := t.getConfiguredSinkChannels()
	for _, ch := range sinkChannels {
		if env.ContainsService(ch, knativeapi.CamelServiceTypeChannel) {
			continue
		}
		channel, err := t.retrieveChannel(e.Integration.Namespace, ch)
		if err != nil {
			return env, err
		}
		hostname := channel.Status.Address.Hostname
		if hostname == "" {
			return env, errors.New("cannot find address of channel " + ch)
		}
		svc := knativeapi.CamelServiceDefinition{
			Name:        ch,
			Host:        hostname,
			Port:        80,
			Protocol:    knativeapi.CamelProtocolHTTP,
			ServiceType: knativeapi.CamelServiceTypeChannel,
			Metadata: map[string]string{
				knativeapi.CamelMetaServicePath: "/",
			},
		}
		env.Services = append(env.Services, svc)
	}
	// Adding default endpoint
	defSvc := knativeapi.CamelServiceDefinition{
		Name:        "default",
		Host:        "0.0.0.0",
		Port:        8080,
		Protocol:    knativeapi.CamelProtocolHTTP,
		ServiceType: knativeapi.CamelServiceTypeEndpoint,
		Metadata: map[string]string{
			knativeapi.CamelMetaServicePath: "/",
		},
	}
	env.Services = append(env.Services, defSvc)
	return env, nil
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

func (*knativeTrait) getSourceChannels(e *Environment) []string {
	channels := make([]string, 0)

	metadata.Each(e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
		channels = append(channels, knativeutil.ExtractChannelNames(meta.FromURIs)...)
		return true
	})

	return channels
}

func (t *knativeTrait) getConfiguredSinkChannels() []string {
	channels := make([]string, 0)
	for _, ch := range strings.Split(t.Sinks, ",") {
		cht := strings.Trim(ch, " \t\"")
		if cht != "" {
			channels = append(channels, cht)
		}
	}
	return channels
}

func (*knativeTrait) getSinkChannels(e *Environment) []string {
	channels := make([]string, 0)

	metadata.Each(e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
		channels = append(channels, knativeutil.ExtractChannelNames(meta.ToURIs)...)
		return true
	})

	return channels
}

func (*knativeTrait) retrieveChannel(namespace string, name string) (*eventing.Channel, error) {
	channel := eventing.Channel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Channel",
			APIVersion: eventing.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	if err := sdk.Get(&channel); err != nil {
		return nil, errors.Wrap(err, "could not retrieve channel "+name+" in namespace "+namespace)
	}
	return &channel, nil
}
