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
	"regexp"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	knativeapi "github.com/apache/camel-k/pkg/apis/camel/v1alpha1/knative"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
)

type knativeTrait struct {
	BaseTrait            `property:",squash"`
	Configuration        string   `property:"configuration"`
	ChannelSources       string   `property:"channel-sources"`
	ChannelSinks         string   `property:"channel-sinks"`
	EndpointSources      string   `property:"endpoint-sources"`
	EndpointSinks        string   `property:"endpoint-sinks"`
	FilterSourceChannels *bool    `property:"filter-source-channels"`
	ChannelAPIs          []string `property:"channel-apis"`
	EndpointAPIs         []string `property:"endpoint-apis"`
	Auto                 *bool    `property:"auto"`
}

const (
	knativeHistoryHeader = "ce-knativehistory"
)

var (
	kindAPIGroupVersionFormat = regexp.MustCompile(`^([^/]+)/([^/]+)/([^/]+)$`)

	defaultChannelAPIs = []string{
		"messaging.knative.dev/v1alpha1/Channel",
		"eventing.knative.dev/v1alpha1/Channel",
		"messaging.knative.dev/v1alpha1/InMemoryChannel",
		"messaging.knative.dev/v1alpha1/KafkaChannel",
		"messaging.knative.dev/v1alpha1/NatssChannel",
	}

	defaultEndpointAPIs = []string{
		"serving.knative.dev/v1beta1/Service",
		"serving.knative.dev/v1alpha1/Service",
		"serving.knative.dev/v1/Service",
	}
)

func init() {
	// Channels are also endpoints
	defaultEndpointAPIs = append(defaultEndpointAPIs, defaultChannelAPIs...)
}

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

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	// Always applying the defaults
	if len(t.ChannelAPIs) == 0 {
		t.ChannelAPIs = append(t.ChannelAPIs, defaultChannelAPIs...)
	}
	if len(t.EndpointAPIs) == 0 {
		t.EndpointAPIs = append(t.EndpointAPIs, defaultEndpointAPIs...)
	}

	if t.Auto == nil || *t.Auto {
		if t.ChannelSources == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.ExtractChannelNames(meta.FromURIs)...)
				return true
			})

			t.ChannelSources = strings.Join(items, ",")
		}
		if t.ChannelSinks == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.ExtractChannelNames(meta.ToURIs)...)
				return true
			})

			t.ChannelSinks = strings.Join(items, ",")
		}
		if t.EndpointSources == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.ExtractEndpointNames(meta.FromURIs)...)
				return true
			})

			t.EndpointSources = strings.Join(items, ",")
		}
		if t.EndpointSinks == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.ExtractEndpointNames(meta.ToURIs)...)
				return true
			})

			t.EndpointSinks = strings.Join(items, ",")
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
	if err := t.createConfiguration(e); err != nil {
		return err
	}
	if err := t.createSubscriptions(e); err != nil {
		return err
	}

	return nil
}

func (t *knativeTrait) createConfiguration(e *Environment) error {
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

	conf, err := env.Serialize()
	if err != nil {
		return errors.Wrap(err, "unable to fetch environment configuration")
	}

	envvar.SetVal(&e.EnvVars, "CAMEL_KNATIVE_CONFIGURATION", conf)

	return nil
}

func (t *knativeTrait) createSubscriptions(e *Environment) error {
	channels := t.extractNames(t.ChannelSources)
	types, err := decodeKindAPIGroupVersions(t.ChannelAPIs)
	if err != nil {
		return err
	}
	for _, ch := range channels {
		chRef, err := knativeutil.GetAddressableReference(t.ctx, t.client, types, e.Integration.Namespace, ch)
		if err != nil {
			return err
		}
		sub := knativeutil.CreateSubscription(*chRef, e.Integration.Name)
		e.Resources.Add(&sub)
	}

	return nil
}

func (t *knativeTrait) configureChannels(e *Environment, env *knativeapi.CamelEnvironment) error {
	sources := t.extractNames(t.ChannelSources)
	sinks := t.extractNames(t.ChannelSinks)

	sr := strset.New(sources...)
	sk := strset.New(sinks...)
	is := strset.Intersection(sr, sk)

	if is.Size() > 0 {
		return fmt.Errorf("cannot use the same channels as source and sink (%s)", is.List())
	}

	types, err := decodeKindAPIGroupVersions(t.ChannelAPIs)
	if err != nil {
		return err
	}

	// Sources
	for _, ch := range sources {
		if env.ContainsService(ch, knativeapi.CamelServiceTypeChannel) {
			continue
		}

		targetURL, err := knativeutil.GetAnySinkURL(t.ctx, t.client, types, e.Integration.Namespace, ch)
		if err != nil && k8serrors.IsNotFound(err) {
			return errors.Errorf("cannot find channel %s", ch)
		} else if err != nil {
			return err
		}

		meta := map[string]string{
			knativeapi.CamelMetaServicePath: "/",
		}
		if t.FilterSourceChannels != nil && *t.FilterSourceChannels {
			meta[knativeapi.CamelMetaFilterHeaderName] = knativeHistoryHeader
			meta[knativeapi.CamelMetaFilterHeaderValue] = targetURL.Host
		}
		svc := knativeapi.CamelServiceDefinition{
			Name:        ch,
			Host:        "0.0.0.0",
			Port:        8080,
			Protocol:    knativeapi.CamelProtocolHTTP,
			ServiceType: knativeapi.CamelServiceTypeChannel,
			Metadata:    meta,
		}
		env.Services = append(env.Services, svc)
	}

	// Sinks
	for _, ch := range sinks {
		if env.ContainsService(ch, knativeapi.CamelServiceTypeChannel) {
			continue
		}

		targetURL, err := knativeutil.GetAnySinkURL(t.ctx, t.client, types, e.Integration.Namespace, ch)
		if err != nil && k8serrors.IsNotFound(err) {
			return errors.Errorf("cannot find channel %s", ch)
		} else if err != nil {
			return err
		}
		t.L.Infof("Found URL for channel %s: %s", ch, targetURL.String())
		svc, err := knativeapi.BuildCamelServiceDefinition(ch, knativeapi.CamelServiceTypeChannel, *targetURL)
		if err != nil {
			return errors.Wrapf(err, "cannot determine address of channel %s", ch)
		}
		env.Services = append(env.Services, svc)
	}

	return nil
}

func (t *knativeTrait) configureEndpoints(e *Environment, env *knativeapi.CamelEnvironment) error {
	sources := t.extractNames(t.EndpointSources)
	sinks := t.extractNames(t.EndpointSinks)

	sr := strset.New(sources...)
	sk := strset.New(sinks...)
	is := strset.Intersection(sr, sk)

	if is.Size() > 0 {
		return fmt.Errorf("cannot use the same enadpoints as source and synk (%s)", is.List())
	}

	types, err := decodeKindAPIGroupVersions(t.EndpointAPIs)
	if err != nil {
		return err
	}

	// Sources
	for _, endpoint := range sources {
		if env.ContainsService(endpoint, knativeapi.CamelServiceTypeEndpoint) {
			continue
		}
		svc := knativeapi.CamelServiceDefinition{
			Name:        endpoint,
			Host:        "0.0.0.0",
			Port:        8080,
			Protocol:    knativeapi.CamelProtocolHTTP,
			ServiceType: knativeapi.CamelServiceTypeEndpoint,
			Metadata: map[string]string{
				knativeapi.CamelMetaServicePath: "/",
			},
		}
		env.Services = append(env.Services, svc)
	}

	// Sinks
	for _, endpoint := range sinks {
		if env.ContainsService(endpoint, knativeapi.CamelServiceTypeEndpoint) {
			continue
		}

		targetURL, err := knativeutil.GetAnySinkURL(t.ctx, t.client, types, e.Integration.Namespace, endpoint)
		if err != nil && k8serrors.IsNotFound(err) {
			return errors.Errorf("cannot find endpoint %s", endpoint)
		} else if err != nil {
			return err
		}
		t.L.Infof("Found URL for endpoint %s: %s", endpoint, targetURL.String())
		svc, err := knativeapi.BuildCamelServiceDefinition(endpoint, knativeapi.CamelServiceTypeEndpoint, *targetURL)
		if err != nil {
			return errors.Wrapf(err, "cannot determine address of endpoint %s", endpoint)
		}
		env.Services = append(env.Services, svc)
	}

	return nil
}

func (t *knativeTrait) extractNames(names string) []string {
	answer := make([]string, 0)
	for _, item := range strings.Split(names, ",") {
		i := strings.Trim(item, " \t\"")
		if i != "" {
			answer = append(answer, i)
		}
	}

	return answer
}

func decodeKindAPIGroupVersions(specs []string) ([]schema.GroupVersionKind, error) {
	lst := make([]schema.GroupVersionKind, 0, len(specs))
	for _, spec := range specs {
		res, err := decodeKindAPIGroupVersion(spec)
		if err != nil {
			return lst, err
		}
		lst = append(lst, res)
	}
	return lst, nil
}

func decodeKindAPIGroupVersion(spec string) (schema.GroupVersionKind, error) {
	if !kindAPIGroupVersionFormat.MatchString(spec) {
		return schema.GroupVersionKind{}, errors.Errorf("spec does not match the Group/Version/Kind format: %s", spec)
	}
	matches := kindAPIGroupVersionFormat.FindStringSubmatch(spec)
	return schema.GroupVersionKind{
		Group:   matches[1],
		Version: matches[2],
		Kind:    matches[3],
	}, nil
}
