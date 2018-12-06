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

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"strconv"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

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
	BaseTrait `property:",squash"`
	Sources   string `property:"sources"`
	Sinks     string `property:"sinks"`
	MinScale  *int   `property:"minScale"`
	MaxScale  *int   `property:"maxScale"`
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
	return nil
}

func (t *knativeTrait) apply(e *Environment) error {
	if err := t.prepareEnvVars(e); err != nil {
		return err
	}
	for _, sub := range t.getSubscriptionsFor(e) {
		e.Resources.Add(sub)
	}
	svc, err := t.getServiceFor(e)
	if err != nil {
		return err
	}
	e.Resources.Add(svc)
	return nil
}

func (t *knativeTrait) prepareEnvVars(e *Environment) error {
	// common env var for Knative integration
	conf, err := t.getConfigurationSerialized(e)
	if err != nil {
		return err
	}
	e.EnvVars["CAMEL_KNATIVE_CONFIGURATION"] = conf
	return nil
}

func (t *knativeTrait) getServiceFor(e *Environment) (*serving.Service, error) {
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
	environment["AB_JOLOKIA_OFF"] = True

	// add env vars from traits
	for k, v := range e.EnvVars {
		environment[k] = v
	}

	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	annotations := make(map[string]string)
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
								Env:   EnvironmentAsEnvVarSlice(environment),
							},
						},
					},
				},
			},
		},
	}

	return &svc, nil
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

	res, err := json.Marshal(env)
	if err != nil {
		return "", errors.Wrap(err, "unable to serialize Knative configuration")
	}
	return string(res), nil
}

func (t *knativeTrait) getConfiguration(e *Environment) (knativeutil.CamelEnvironment, error) {
	env := knativeutil.NewCamelEnvironment()
	// Sources
	sourceChannels := t.getConfiguredSourceChannels()
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
	// Sinks
	sinkChannels := t.getConfiguredSinkChannels()
	for _, ch := range sinkChannels {
		channel, err := t.retrieveChannel(e.Integration.Namespace, ch)
		if err != nil {
			return env, err
		}
		hostname := channel.Status.Address.Hostname
		if hostname == "" {
			return env, errors.New("cannot find address of channel " + ch)
		}
		svc := knativeutil.CamelServiceDefinition{
			Name:        ch,
			Host:        hostname,
			Port:        80,
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
