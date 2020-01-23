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
	"net/url"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	knativeapi "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/envvar"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
	serving "knative.dev/serving/pkg/apis/serving/v1"
)

// The Knative trait automatically discovers addresses of Knative resources and inject them into the
// running integration.
//
// The full Knative configuration is injected in the CAMEL_KNATIVE_CONFIGURATION in JSON format.
// The Camel Knative component will then use the full configuration to configure the routes.
//
// The trait is enabled by default when the Knative profile is active.
//
// +camel-k:trait=knative
type knativeTrait struct {
	BaseTrait `property:",squash"`
	// Can be used to inject a Knative complete configuration in JSON format.
	Configuration string `property:"configuration"`
	// Comma-separated list of channels used as source of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSources string `property:"channel-sources"`
	// Comma-separated list of channels used as destination of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSinks string `property:"channel-sinks"`
	// Comma-separated list of channels used as source of integration routes.
	EndpointSources string `property:"endpoint-sources"`
	// Comma-separated list of endpoints used as destination of integration routes.
	// Can contain simple endpoint names or full Camel URIs.
	EndpointSinks string `property:"endpoint-sinks"`
	// Comma-separated list of event types that the integration will be subscribed to.
	// Can contain simple event types or full Camel URIs (to use a specific broker different from "default").
	EventSources string `property:"event-sources"`
	// Comma-separated list of event types that the integration will produce.
	// Can contain simple event types or full Camel URIs (to use a specific broker).
	EventSinks string `property:"event-sinks"`
	// Enables filtering on events based on the header "ce-knativehistory". Since this is an experimental header
	// that can be removed in a future version of Knative, filtering is enabled only when the integration is
	// listening from more than 1 channel.
	FilterSourceChannels *bool `property:"filter-source-channels"`
	// Enable automatic discovery of all trait properties.
	Auto *bool `property:"auto"`
}

const (
	knativeHistoryHeader = "ce-knativehistory"
)

func newKnativeTrait() *knativeTrait {
	t := &knativeTrait{
		BaseTrait: newBaseTrait("knative"),
	}

	return t
}

func (t *knativeTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if t.ChannelSources == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.FromURIs, knativeapi.CamelServiceTypeChannel)...)
				return true
			})

			t.ChannelSources = strings.Join(items, ",")
		}
		if t.ChannelSinks == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.ToURIs, knativeapi.CamelServiceTypeChannel)...)
				return true
			})

			t.ChannelSinks = strings.Join(items, ",")
		}
		if t.EndpointSources == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.FromURIs, knativeapi.CamelServiceTypeEndpoint)...)
				return true
			})

			t.EndpointSources = strings.Join(items, ",")
		}
		if t.EndpointSinks == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.ToURIs, knativeapi.CamelServiceTypeEndpoint)...)
				return true
			})

			t.EndpointSinks = strings.Join(items, ",")
		}
		if t.EventSources == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.FromURIs, knativeapi.CamelServiceTypeEvent)...)
				return true
			})

			t.EventSources = strings.Join(items, ",")
		}
		if t.EventSinks == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.ToURIs, knativeapi.CamelServiceTypeEvent)...)
				return true
			})

			t.EventSinks = strings.Join(items, ",")
		}
		if len(strings.Split(t.ChannelSources, ",")) > 1 {
			// Always filter channels when the integration subscribes to more than one
			// Using Knative experimental header: https://github.com/knative/eventing/blob/7df0cc56c28d58223ff25d5ddfb487fa8c29a004/pkg/provisioners/message.go#L28
			// TODO: filter automatically all source channels when the feature becomes stable
			filter := true
			t.FilterSourceChannels = &filter
		}
	}

	return true, nil
}

func (t *knativeTrait) Apply(e *Environment) error {
	env := knativeapi.NewCamelEnvironment()
	if t.Configuration != "" {
		if err := env.Deserialize(t.Configuration); err != nil {
			return err
		}
	}

	if err := t.configureChannels(e, &env); err != nil {
		return err
	}
	if err := t.configureEndpoints(e, &env); err != nil {
		return err
	}
	if err := t.configureEvents(e, &env); err != nil {
		return err
	}

	conf, err := env.Serialize()
	if err != nil {
		return errors.Wrap(err, "unable to fetch environment configuration")
	}

	envvar.SetVal(&e.EnvVars, "CAMEL_KNATIVE_CONFIGURATION", conf)

	return nil
}

func (t *knativeTrait) configureChannels(e *Environment, env *knativeapi.CamelEnvironment) error {
	// Sources
	err := t.ifServiceMissingDo(e, env, t.ChannelSources, knativeapi.CamelServiceTypeChannel, knativeapi.CamelEndpointKindSource,
		func(ref *corev1.ObjectReference, loc *url.URL, serviceURI string) error {
			meta := map[string]string{
				knativeapi.CamelMetaServicePath:       "/",
				knativeapi.CamelMetaEndpointKind:      string(knativeapi.CamelEndpointKindSource),
				knativeapi.CamelMetaKnativeAPIVersion: ref.APIVersion,
				knativeapi.CamelMetaKnativeKind:       ref.Kind,
			}
			if t.FilterSourceChannels != nil && *t.FilterSourceChannels {
				meta[knativeapi.CamelMetaFilterPrefix+knativeHistoryHeader] = loc.Host
			}
			svc := knativeapi.CamelServiceDefinition{
				Name:        ref.Name,
				Host:        "0.0.0.0",
				Port:        8080,
				ServiceType: knativeapi.CamelServiceTypeChannel,
				Metadata:    meta,
			}
			env.Services = append(env.Services, svc)

			if err := t.createSubscription(e, ref); err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Sinks
	err = t.ifServiceMissingDo(e, env, t.ChannelSinks, knativeapi.CamelServiceTypeChannel, knativeapi.CamelEndpointKindSink,
		func(ref *corev1.ObjectReference, loc *url.URL, serviceURI string) error {
			svc, err := knativeapi.BuildCamelServiceDefinition(ref.Name, knativeapi.CamelEndpointKindSink,
				knativeapi.CamelServiceTypeChannel, *loc, ref.APIVersion, ref.Kind)
			if err != nil {
				return err
			}
			env.Services = append(env.Services, svc)
			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

func (t *knativeTrait) createSubscription(e *Environment, ref *corev1.ObjectReference) error {
	sub := knativeutil.CreateSubscription(*ref, e.Integration.Name)
	e.Resources.Add(sub)
	return nil
}

func (t *knativeTrait) configureEndpoints(e *Environment, env *knativeapi.CamelEnvironment) error {
	// Sources
	serviceSources := t.extractServices(t.EndpointSources, knativeapi.CamelServiceTypeEndpoint)
	for _, endpoint := range serviceSources {
		ref, err := knativeutil.ExtractObjectReference(endpoint)
		if err != nil {
			return err
		}
		if env.ContainsService(endpoint, knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEndpoint,
			serving.SchemeGroupVersion.String(), "Service") {
			continue
		}
		svc := knativeapi.CamelServiceDefinition{
			Name:        ref.Name,
			Host:        "0.0.0.0",
			Port:        8080,
			ServiceType: knativeapi.CamelServiceTypeEndpoint,
			Metadata: map[string]string{
				knativeapi.CamelMetaServicePath:       "/",
				knativeapi.CamelMetaEndpointKind:      string(knativeapi.CamelEndpointKindSource),
				knativeapi.CamelMetaKnativeAPIVersion: serving.SchemeGroupVersion.String(),
				knativeapi.CamelMetaKnativeKind:       "Service",
			},
		}
		env.Services = append(env.Services, svc)
	}

	// Sinks
	err := t.ifServiceMissingDo(e, env, t.EndpointSinks, knativeapi.CamelServiceTypeEndpoint, knativeapi.CamelEndpointKindSink,
		func(ref *corev1.ObjectReference, loc *url.URL, serviceURI string) error {
			svc, err := knativeapi.BuildCamelServiceDefinition(ref.Name, knativeapi.CamelEndpointKindSink,
				knativeapi.CamelServiceTypeEndpoint, *loc, ref.APIVersion, ref.Kind)
			if err != nil {
				return err
			}
			env.Services = append(env.Services, svc)
			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

func (t *knativeTrait) configureEvents(e *Environment, env *knativeapi.CamelEnvironment) error {
	// Sources
	err := t.withServiceDo(false, e, env, t.EventSources, knativeapi.CamelServiceTypeEvent, knativeapi.CamelEndpointKindSource,
		func(ref *corev1.ObjectReference, loc *url.URL, serviceURI string) error {
			// Iterate over all, without skipping duplicates
			eventType := knativeutil.ExtractEventType(serviceURI)
			t.createTrigger(e, ref, eventType)

			if !env.ContainsService(ref.Name, knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEvent, ref.APIVersion, ref.Kind) {
				svc := knativeapi.CamelServiceDefinition{
					Name:        ref.Name,
					Host:        "0.0.0.0",
					Port:        8080,
					ServiceType: knativeapi.CamelServiceTypeEvent,
					Metadata: map[string]string{
						knativeapi.CamelMetaServicePath:       "/",
						knativeapi.CamelMetaEndpointKind:      string(knativeapi.CamelEndpointKindSource),
						knativeapi.CamelMetaKnativeAPIVersion: ref.APIVersion,
						knativeapi.CamelMetaKnativeKind:       ref.Kind,
					},
				}
				env.Services = append(env.Services, svc)
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Sinks
	err = t.ifServiceMissingDo(e, env, t.EventSinks, knativeapi.CamelServiceTypeEvent, knativeapi.CamelEndpointKindSink,
		func(ref *corev1.ObjectReference, loc *url.URL, serviceURI string) error {
			svc, err := knativeapi.BuildCamelServiceDefinition(ref.Name, knativeapi.CamelEndpointKindSink,
				knativeapi.CamelServiceTypeEvent, *loc, ref.APIVersion, ref.Kind)
			if err != nil {
				return err
			}
			env.Services = append(env.Services, svc)
			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

func (t *knativeTrait) createTrigger(e *Environment, ref *corev1.ObjectReference, eventType string) {
	// TODO extend to additional filters too, to filter them at source and not at destination
	found := e.Resources.HasKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Spec.Broker == ref.Name &&
			trigger.Spec.Filter != nil &&
			trigger.Spec.Filter.Attributes != nil &&
			(*trigger.Spec.Filter.Attributes)["type"] == eventType
	})
	if !found {
		trigger := knativeutil.CreateTrigger(*ref, e.Integration.Name, eventType)
		e.Resources.Add(trigger)
	}
}

func (t *knativeTrait) ifServiceMissingDo(
	e *Environment,
	env *knativeapi.CamelEnvironment,
	serviceURIsAsString string,
	serviceType knativeapi.CamelServiceType,
	endpointKind knativeapi.CamelEndpointKind,
	gen func(ref *corev1.ObjectReference, url *url.URL, serviceURI string) error) error {
	return t.withServiceDo(true, e, env, serviceURIsAsString, serviceType, endpointKind, gen)
}

func (t *knativeTrait) withServiceDo(
	skipDuplicates bool,
	e *Environment,
	env *knativeapi.CamelEnvironment,
	serviceURIsAsString string,
	serviceType knativeapi.CamelServiceType,
	endpointKind knativeapi.CamelEndpointKind,
	gen func(ref *corev1.ObjectReference, url *url.URL, serviceURI string) error) error {

	serviceURIs := t.extractServices(serviceURIsAsString, serviceType)
	for _, serviceURI := range serviceURIs {
		ref, err := knativeutil.ExtractObjectReference(serviceURI)
		if err != nil {
			return err
		}
		if skipDuplicates && env.ContainsService(ref.Name, endpointKind, serviceType, ref.APIVersion, ref.Kind) {
			continue
		}
		possibleRefs := knativeutil.FillMissingReferenceData(serviceType, ref)
		actualRef, err := knativeutil.GetAddressableReference(t.ctx, t.client, possibleRefs, e.Integration.Namespace, ref.Name)
		if err != nil && k8serrors.IsNotFound(err) {
			return errors.Errorf("cannot find %s %s", serviceType, ref.Name)
		} else if err != nil {
			return errors.Wrapf(err, "error looking up %s %s", serviceType, ref.Name)
		}
		targetURL, err := knativeutil.GetSinkURL(t.ctx, t.client, actualRef, e.Integration.Namespace)
		if err != nil {
			return errors.Wrapf(err, "cannot determine address of %s %s", string(serviceType), ref.Name)
		}
		t.L.Infof("Found URL for %s: %s", string(serviceType), targetURL.String())
		err = gen(actualRef, targetURL, serviceURI)
		if err != nil {
			return errors.Wrapf(err, "unexpected error while executing handler for %s %s", string(serviceType), ref.Name)
		}
	}
	return nil
}

func (t *knativeTrait) extractServices(names string, serviceType knativeapi.CamelServiceType) []string {
	answer := make([]string, 0)
	for _, item := range strings.Split(names, ",") {
		i := strings.Trim(item, " \t\"")
		if i != "" {
			i = knativeutil.NormalizeToURI(serviceType, i)
			answer = append(answer, i)
		}
	}
	return answer
}
