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
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	eventing "knative.dev/eventing/pkg/apis/eventing/v1beta1"
	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	knativeapi "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
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
	Configuration string `property:"configuration" json:"configuration,omitempty"`
	// List of channels used as source of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSources []string `property:"channel-sources" json:"channelSources,omitempty"`
	// List of channels used as destination of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSinks []string `property:"channel-sinks" json:"channelSinks,omitempty"`
	// List of channels used as source of integration routes.
	EndpointSources []string `property:"endpoint-sources" json:"endpointSources,omitempty"`
	// List of endpoints used as destination of integration routes.
	// Can contain simple endpoint names or full Camel URIs.
	EndpointSinks []string `property:"endpoint-sinks" json:"endpointSinks,omitempty"`
	// List of event types that the integration will be subscribed to.
	// Can contain simple event types or full Camel URIs (to use a specific broker different from "default").
	EventSources []string `property:"event-sources" json:"eventSources,omitempty"`
	// List of event types that the integration will produce.
	// Can contain simple event types or full Camel URIs (to use a specific broker).
	EventSinks []string `property:"event-sinks" json:"eventSinks,omitempty"`
	// Enables filtering on events based on the header "ce-knativehistory". Since this header has been removed in newer versions of
	// Knative, filtering is disabled by default.
	FilterSourceChannels *bool `property:"filter-source-channels" json:"filterSourceChannels,omitempty"`
	// Enables Knative CamelSource pre 0.15 compatibility fixes (will be removed in future versions).
	CamelSourceCompat *bool `property:"camel-source-compat" json:"camelSourceCompat,omitempty"`
	// Allows binding the integration to a sink via a Knative SinkBinding resource.
	// This can be used when the integration targets a single sink.
	// It's enabled by default when the integration targets a single sink
	// (except when the integration is owned by a Knative source).
	SinkBinding *bool `property:"sink-binding" json:"sinkBinding,omitempty"`
	// Enable automatic discovery of all trait properties.
	Auto *bool `property:"auto" json:"auto,omitempty"`
}

const (
	knativeHistoryHeader = "ce-knativehistory"
)

func newKnativeTrait() Trait {
	t := &knativeTrait{
		BaseTrait: NewBaseTrait("knative", 400),
	}

	return t
}

// IsAllowedInProfile overrides default
func (t *knativeTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile == v1.TraitProfileKnative
}

func (t *knativeTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		if len(t.ChannelSources) == 0 {
			items := make([]string, 0)
			sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}
			metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.FromURIs, knativeapi.CamelServiceTypeChannel)...)
				return true
			})

			t.ChannelSources = items
		}
		if len(t.ChannelSinks) == 0 {
			items := make([]string, 0)
			sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}
			metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.ToURIs, knativeapi.CamelServiceTypeChannel)...)
				return true
			})

			t.ChannelSinks = items
		}
		if len(t.EndpointSources) == 0 {
			items := make([]string, 0)
			sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}
			metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.FromURIs, knativeapi.CamelServiceTypeEndpoint)...)
				return true
			})

			t.EndpointSources = items
		}
		if len(t.EndpointSinks) == 0 {
			items := make([]string, 0)
			sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}
			metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.ToURIs, knativeapi.CamelServiceTypeEndpoint)...)
				return true
			})

			t.EndpointSinks = items
		}
		if len(t.EventSources) == 0 {
			items := make([]string, 0)
			sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}
			metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.FromURIs, knativeapi.CamelServiceTypeEvent)...)
				return true
			})

			t.EventSources = items
		}
		if len(t.EventSinks) == 0 {
			items := make([]string, 0)
			sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}
			metadata.Each(e.CamelCatalog, sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.FilterURIs(meta.ToURIs, knativeapi.CamelServiceTypeEvent)...)
				return true
			})

			t.EventSinks = items
		}
		if t.FilterSourceChannels == nil {
			// Filtering is no longer used by default
			filter := false
			t.FilterSourceChannels = &filter
		}
		if t.SinkBinding == nil {
			allowed := t.isSinkBindingAllowed(e)
			t.SinkBinding = &allowed
		}
	}

	return true, nil
}

func (t *knativeTrait) Apply(e *Environment) error {
	// To be removed when Knative CamelSources < 0.15 will no longer be supported
	// Older versions of Knative Sources use a loader rather than an interceptor
	if t.CamelSourceCompat == nil || *t.CamelSourceCompat {
		for i, s := range e.Integration.Spec.Sources {
			if s.Loader == "knative-source" {
				s.Loader = ""
				util.StringSliceUniqueAdd(&s.Interceptors, "knative-source")
				e.Integration.Spec.Sources[i] = s
			}
		}
	}
	// End of temporary code

	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// Interceptor may have been set by a Knative CamelSource
		if util.StringSliceExists(e.getAllInterceptors(), "knative-source") {
			// Adding required libraries for Camel sources
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.k:camel-knative")
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.k:camel-k-knative")
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.k:camel-k-knative-producer")
		}
	}

	if t.SinkBinding != nil && *t.SinkBinding {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.k:camel-k-knative")
	}

	if len(t.ChannelSources) > 0 || len(t.EndpointSources) > 0 || len(t.EventSources) > 0 {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityPlatformHTTP)
	}
	if len(t.ChannelSinks) > 0 || len(t.EndpointSinks) > 0 || len(t.EventSinks) > 0 {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityPlatformHTTP)
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		env := knativeapi.NewCamelEnvironment()
		if t.Configuration != "" {
			if err := env.Deserialize(t.Configuration); err != nil {
				return err
			}
		}

		// Convert deprecated Host and Port fields to URL field
		// Can be removed once CamelSource controller migrate to the new API
		for i, service := range env.Services {
			if service.URL == "" {
				URL := "http://" + service.Host
				if service.Port != nil {
					URL = URL + ":" + strconv.Itoa(*service.Port)
				}
				service.URL = URL
				service.Host = ""
				service.Port = nil
				env.Services[i] = service
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
		if err := t.configureSinkBinding(e, &env); err != nil {
			return err
		}

		conf, err := env.Serialize()
		if err != nil {
			return errors.Wrap(err, "unable to fetch environment configuration")
		}

		envvar.SetVal(&e.EnvVars, "CAMEL_KNATIVE_CONFIGURATION", conf)
	}

	return nil
}

func (t *knativeTrait) configureChannels(e *Environment, env *knativeapi.CamelEnvironment) error {
	// Sources
	err := t.ifServiceMissingDo(e, env, t.ChannelSources, knativeapi.CamelServiceTypeChannel, knativeapi.CamelEndpointKindSource,
		func(ref *corev1.ObjectReference, serviceURI string, urlProvider func() (*url.URL, error)) error {
			loc, err := urlProvider()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/channels/%s", ref.Name)
			meta := map[string]string{
				knativeapi.CamelMetaEndpointKind:      string(knativeapi.CamelEndpointKindSource),
				knativeapi.CamelMetaKnativeAPIVersion: ref.APIVersion,
				knativeapi.CamelMetaKnativeKind:       ref.Kind,
				knativeapi.CamelMetaKnativeReply:      "false",
			}
			if t.FilterSourceChannels != nil && *t.FilterSourceChannels {
				meta[knativeapi.CamelMetaFilterPrefix+knativeHistoryHeader] = loc.Host
			}
			svc := knativeapi.CamelServiceDefinition{
				Name:        ref.Name,
				ServiceType: knativeapi.CamelServiceTypeChannel,
				Path:        path,
				Metadata:    meta,
			}
			env.Services = append(env.Services, svc)

			if err := t.createSubscription(e, ref, path); err != nil {
				return err
			}

			return nil
		})
	if err != nil {
		return err
	}

	if t.SinkBinding == nil || !*t.SinkBinding {
		// Sinks
		err = t.ifServiceMissingDo(e, env, t.ChannelSinks, knativeapi.CamelServiceTypeChannel, knativeapi.CamelEndpointKindSink,
			func(ref *corev1.ObjectReference, serviceURI string, urlProvider func() (*url.URL, error)) error {
				loc, err := urlProvider()
				if err != nil {
					return err
				}
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
	}

	return nil
}

func (t *knativeTrait) createSubscription(e *Environment, ref *corev1.ObjectReference, path string) error {
	if ref.Namespace == "" {
		ref.Namespace = e.Integration.Namespace
	}
	sub := knativeutil.CreateSubscription(*ref, e.Integration.Name, path)
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
			ServiceType: knativeapi.CamelServiceTypeEndpoint,
			Path:        "/",
			Metadata: map[string]string{
				knativeapi.CamelMetaEndpointKind:      string(knativeapi.CamelEndpointKindSource),
				knativeapi.CamelMetaKnativeAPIVersion: serving.SchemeGroupVersion.String(),
				knativeapi.CamelMetaKnativeKind:       "Service",
				// knative.reply is left to default ("true") in case of simple service
			},
		}
		env.Services = append(env.Services, svc)
	}

	// Sinks
	if t.SinkBinding == nil || !*t.SinkBinding {
		err := t.ifServiceMissingDo(e, env, t.EndpointSinks, knativeapi.CamelServiceTypeEndpoint, knativeapi.CamelEndpointKindSink,
			func(ref *corev1.ObjectReference, serviceURI string, urlProvider func() (*url.URL, error)) error {
				loc, err := urlProvider()
				if err != nil {
					return err
				}
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
	}

	return nil
}

func (t *knativeTrait) configureEvents(e *Environment, env *knativeapi.CamelEnvironment) error {
	// Sources
	err := t.withServiceDo(false, e, env, t.EventSources, knativeapi.CamelServiceTypeEvent, knativeapi.CamelEndpointKindSource,
		func(ref *corev1.ObjectReference, serviceURI string, _ func() (*url.URL, error)) error {
			// Iterate over all, without skipping duplicates
			eventType := knativeutil.ExtractEventType(serviceURI)
			serviceName := eventType
			if serviceName == "" {
				serviceName = "default"
			}
			servicePath := fmt.Sprintf("/events/%s", eventType)
			t.createTrigger(e, ref, eventType, servicePath)

			if !env.ContainsService(serviceName, knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEvent, ref.APIVersion, ref.Kind) {
				svc := knativeapi.CamelServiceDefinition{
					Name:        serviceName,
					ServiceType: knativeapi.CamelServiceTypeEvent,
					Path:        servicePath,
					Metadata: map[string]string{
						knativeapi.CamelMetaEndpointKind:      string(knativeapi.CamelEndpointKindSource),
						knativeapi.CamelMetaKnativeAPIVersion: ref.APIVersion,
						knativeapi.CamelMetaKnativeKind:       ref.Kind,
						knativeapi.CamelMetaKnativeReply:      "false",
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
	if t.SinkBinding == nil || !*t.SinkBinding {
		err = t.ifServiceMissingDo(e, env, t.EventSinks, knativeapi.CamelServiceTypeEvent, knativeapi.CamelEndpointKindSink,
			func(ref *corev1.ObjectReference, serviceURI string, urlProvider func() (*url.URL, error)) error {
				loc, err := urlProvider()
				if err != nil {
					return err
				}
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
	}

	return nil
}

func (t *knativeTrait) isSinkBindingAllowed(e *Environment) bool {
	services := t.extractServices(t.ChannelSinks, knativeapi.CamelServiceTypeChannel)
	services = append(services, t.extractServices(t.EndpointSinks, knativeapi.CamelServiceTypeEndpoint)...)
	services = append(services, t.extractServices(t.EventSinks, knativeapi.CamelServiceTypeEvent)...)

	if len(services) != 1 {
		return false
	}

	for _, owner := range e.Integration.OwnerReferences {
		if strings.Contains(owner.APIVersion, "sources.knative.dev") {
			return false
		}
	}
	return true
}

func (t *knativeTrait) configureSinkBinding(e *Environment, env *knativeapi.CamelEnvironment) error {
	if t.SinkBinding == nil || !*t.SinkBinding {
		return nil
	}
	var serviceType knativeapi.CamelServiceType
	services := t.extractServices(t.ChannelSinks, knativeapi.CamelServiceTypeChannel)
	if len(services) > 0 {
		serviceType = knativeapi.CamelServiceTypeChannel
	}
	services = append(services, t.extractServices(t.EndpointSinks, knativeapi.CamelServiceTypeEndpoint)...)
	if len(serviceType) == 0 && len(services) > 0 {
		serviceType = knativeapi.CamelServiceTypeEndpoint
	}
	services = append(services, t.extractServices(t.EventSinks, knativeapi.CamelServiceTypeEvent)...)
	if len(serviceType) == 0 && len(services) > 0 {
		serviceType = knativeapi.CamelServiceTypeEvent
	}

	if len(services) != 1 {
		return fmt.Errorf("sinkbinding can only be used with a single sink: found %d sinks", len(services))
	}

	err := t.withServiceDo(false, e, env, services, serviceType, knativeapi.CamelEndpointKindSink, func(ref *corev1.ObjectReference, serviceURI string, _ func() (*url.URL, error)) error {
		e.ApplicationProperties["camel.k.customizer.sinkbinding.enabled"] = "true"
		e.ApplicationProperties["camel.k.customizer.sinkbinding.name"] = ref.Name
		e.ApplicationProperties["camel.k.customizer.sinkbinding.type"] = string(serviceType)
		e.ApplicationProperties["camel.k.customizer.sinkbinding.kind"] = ref.Kind
		e.ApplicationProperties["camel.k.customizer.sinkbinding.api-version"] = ref.APIVersion

		if e.IntegrationInPhase(v1.IntegrationPhaseDeploying) {
			e.PostStepProcessors = append(e.PostStepProcessors, func(e *Environment) error {
				sinkBindingInjected := false
				e.Resources.Visit(func(object runtime.Object) {
					gvk := object.GetObjectKind().GroupVersionKind()
					if gvk.Kind == "SinkBinding" && strings.Contains(gvk.Group, "knative") {
						sinkBindingInjected = true
					}
				})
				if sinkBindingInjected {
					return nil
				}

				controller := e.Resources.GetController(func(object runtime.Object) bool {
					return true
				})
				if controller != nil && !reflect.ValueOf(controller).IsNil() {
					gvk := controller.GetObjectKind().GroupVersionKind()
					av, k := gvk.ToAPIVersionAndKind()
					source := corev1.ObjectReference{
						Kind:       k,
						Namespace:  e.Integration.Namespace,
						Name:       e.Integration.Name,
						APIVersion: av,
					}
					target := corev1.ObjectReference{
						Kind:       ref.Kind,
						Namespace:  e.Integration.Namespace,
						Name:       ref.Name,
						APIVersion: ref.APIVersion,
					}
					e.Resources.AddFirst(knativeutil.CreateSinkBinding(source, target))
				}
				return nil
			})
		}
		return nil
	})

	return err
}

func (t *knativeTrait) createTrigger(e *Environment, ref *corev1.ObjectReference, eventType string, path string) {
	// TODO extend to additional filters too, to filter them at source and not at destination
	found := e.Resources.HasKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Spec.Broker == ref.Name &&
			trigger.Spec.Filter != nil &&
			trigger.Spec.Filter.Attributes["type"] == eventType // can be also missing
	})
	if !found {
		if ref.Namespace == "" {
			ref.Namespace = e.Integration.Namespace
		}
		trigger := knativeutil.CreateTrigger(*ref, e.Integration.Name, eventType, path)
		e.Resources.Add(trigger)
	}
}

func (t *knativeTrait) ifServiceMissingDo(
	e *Environment,
	env *knativeapi.CamelEnvironment,
	serviceURIs []string,
	serviceType knativeapi.CamelServiceType,
	endpointKind knativeapi.CamelEndpointKind,
	gen func(ref *corev1.ObjectReference, serviceURI string, urlProvider func() (*url.URL, error)) error) error {
	return t.withServiceDo(true, e, env, serviceURIs, serviceType, endpointKind, gen)
}

func (t *knativeTrait) withServiceDo(
	skipDuplicates bool,
	e *Environment,
	env *knativeapi.CamelEnvironment,
	serviceURIs []string,
	serviceType knativeapi.CamelServiceType,
	endpointKind knativeapi.CamelEndpointKind,
	gen func(ref *corev1.ObjectReference, serviceURI string, urlProvider func() (*url.URL, error)) error) error {

	for _, serviceURI := range t.extractServices(serviceURIs, serviceType) {
		ref, err := knativeutil.ExtractObjectReference(serviceURI)
		if err != nil {
			return err
		}
		if skipDuplicates && env.ContainsService(ref.Name, endpointKind, serviceType, ref.APIVersion, ref.Kind) {
			continue
		}
		possibleRefs := knativeutil.FillMissingReferenceData(serviceType, ref)
		var actualRef *corev1.ObjectReference
		if len(possibleRefs) == 1 {
			actualRef = &possibleRefs[0]
		} else {
			actualRef, err = knativeutil.GetAddressableReference(t.Ctx, t.Client, possibleRefs, e.Integration.Namespace, ref.Name)
			if err != nil && k8serrors.IsNotFound(err) {
				return errors.Errorf("cannot find %s", serviceType.ResourceDescription(ref.Name))
			} else if err != nil {
				return errors.Wrapf(err, "error looking up %s", serviceType.ResourceDescription(ref.Name))
			}
		}

		urlProvider := func() (*url.URL, error) {
			targetURL, err := knativeutil.GetSinkURL(t.Ctx, t.Client, actualRef, e.Integration.Namespace)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot determine address of %s", serviceType.ResourceDescription(ref.Name))
			}
			t.L.Infof("Found URL for %s: %s", serviceType.ResourceDescription(ref.Name), targetURL.String())
			return targetURL, nil
		}

		err = gen(actualRef, serviceURI, urlProvider)
		if err != nil {
			return errors.Wrapf(err, "unexpected error while executing handler for %s", serviceType.ResourceDescription(ref.Name))
		}
	}
	return nil
}

func (t *knativeTrait) extractServices(names []string, serviceType knativeapi.CamelServiceType) []string {
	answer := make([]string, 0)
	for _, item := range names {
		i := strings.Trim(item, " \t\"")
		if i != "" {
			i = knativeutil.NormalizeToURI(serviceType, i)
			answer = append(answer, i)
		}
	}
	sort.Strings(answer)
	return answer
}
