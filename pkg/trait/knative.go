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
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	eventing "knative.dev/eventing/pkg/apis/eventing/v1"
	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	knativeapi "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/envvar"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type knativeTrait struct {
	BaseTrait
	traitv1.KnativeTrait `property:",squash"`
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

// IsAllowedInProfile overrides default.
func (t *knativeTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile.Equal(v1.TraitProfileKnative)
}

func (t *knativeTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if pointer.BoolDeref(t.Auto, true) {
		if len(t.ChannelSources) == 0 {
			items := make([]string, 0)
			sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
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
			sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
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
			sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
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
			sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
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
			sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
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
			sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
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
			t.FilterSourceChannels = pointer.Bool(false)
		}
		if t.SinkBinding == nil {
			allowed := t.isSinkBindingAllowed(e)
			t.SinkBinding = &allowed
		}
	}

	return true, nil
}

func (t *knativeTrait) Apply(e *Environment) error {
	if pointer.BoolDeref(t.SinkBinding, false) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel.k:camel-k-knative")
	}

	if len(t.ChannelSources) > 0 || len(t.EndpointSources) > 0 || len(t.EventSources) > 0 {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityPlatformHTTP)
	}
	if len(t.ChannelSinks) > 0 || len(t.EndpointSinks) > 0 || len(t.EventSinks) > 0 {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityPlatformHTTP)
	}

	if e.IntegrationInRunningPhases() {
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
	err := t.ifServiceMissingDo(e, env, t.ChannelSources, knativeapi.CamelServiceTypeChannel,
		knativeapi.CamelEndpointKindSource,
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
			if pointer.BoolDeref(t.FilterSourceChannels, false) {
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

	if !pointer.BoolDeref(t.SinkBinding, false) {
		// Sinks
		err = t.ifServiceMissingDo(e, env, t.ChannelSinks, knativeapi.CamelServiceTypeChannel,
			knativeapi.CamelEndpointKindSink,
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
	if !pointer.BoolDeref(t.SinkBinding, false) {
		err := t.ifServiceMissingDo(e, env, t.EndpointSinks, knativeapi.CamelServiceTypeEndpoint,
			knativeapi.CamelEndpointKindSink,
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
	err := t.withServiceDo(false, e, env, t.EventSources,
		knativeapi.CamelServiceTypeEvent, knativeapi.CamelEndpointKindSource,
		func(ref *corev1.ObjectReference, serviceURI string, _ func() (*url.URL, error)) error {
			// Iterate over all, without skipping duplicates
			eventType := knativeutil.ExtractEventType(serviceURI)
			serviceName := eventType
			if serviceName == "" {
				serviceName = "default"
			}
			servicePath := fmt.Sprintf("/events/%s", eventType)
			t.createTrigger(e, ref, eventType, servicePath)

			if !env.ContainsService(serviceName, knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEvent,
				ref.APIVersion, ref.Kind) {
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
	if !pointer.BoolDeref(t.SinkBinding, false) {
		err = t.ifServiceMissingDo(e, env, t.EventSinks, knativeapi.CamelServiceTypeEvent,
			knativeapi.CamelEndpointKindSink,
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
	if !pointer.BoolDeref(t.SinkBinding, false) {
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

	err := t.withServiceDo(false, e, env, services, serviceType, knativeapi.CamelEndpointKindSink, func(
		ref *corev1.ObjectReference, serviceURI string, _ func() (*url.URL, error)) error {
		e.ApplicationProperties["camel.k.customizer.sinkbinding.enabled"] = "true"
		e.ApplicationProperties["camel.k.customizer.sinkbinding.name"] = ref.Name
		e.ApplicationProperties["camel.k.customizer.sinkbinding.type"] = string(serviceType)
		e.ApplicationProperties["camel.k.customizer.sinkbinding.kind"] = ref.Kind
		e.ApplicationProperties["camel.k.customizer.sinkbinding.api-version"] = ref.APIVersion

		if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
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

				controller := e.Resources.GetController(func(object ctrl.Object) bool {
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

					// Add the SinkBinding in first position, to make sure it is created
					// before the reference source, so that the SinkBinding webhook has
					// all the information to perform injection.
					e.Resources.AddFirst(knativeutil.CreateSinkBinding(source, target))

					// Make sure the Eventing webhook will select the source resource,
					// in order to inject the sink information.
					// This is necessary for Knative environments, that are configured
					// with SINK_BINDING_SELECTION_MODE=inclusion.
					// See:
					// https://knative.dev/v0.20-docs/eventing/sources/sinkbinding/
					// https://github.com/knative/operator/blob/c60e62bb86ff318c44d1520927d2182659cfdeb5/docs/configuration.md#specsinkbindingselectionmode
					controller.GetLabels()["bindings.knative.dev/include"] = "true"
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
			actualRef, err = knativeutil.GetAddressableReference(e.Ctx, t.Client, possibleRefs,
				e.Integration.Namespace, ref.Name)
			if err != nil && k8serrors.IsNotFound(err) {
				return errors.Errorf("cannot find %s", serviceType.ResourceDescription(ref.Name))
			} else if err != nil {
				return errors.Wrapf(err, "error looking up %s", serviceType.ResourceDescription(ref.Name))
			}
		}

		urlProvider := func() (*url.URL, error) {
			targetURL, err := knativeutil.GetSinkURL(e.Ctx, t.Client, actualRef, e.Integration.Namespace)
			if err != nil {
				return nil, errors.Wrapf(err,
					"cannot determine address of %s",
					serviceType.ResourceDescription(ref.Name))
			}
			t.L.Infof("Found URL for %s: %s", serviceType.ResourceDescription(ref.Name), targetURL.String())
			return targetURL, nil
		}

		err = gen(actualRef, serviceURI, urlProvider)
		if err != nil {
			return errors.Wrapf(err,
				"unexpected error while executing handler for %s",
				serviceType.ResourceDescription(ref.Name))
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
