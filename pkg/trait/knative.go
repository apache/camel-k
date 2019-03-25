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
	"strings"

	"github.com/pkg/errors"

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/envvar"

	knativeapi "github.com/apache/camel-k/pkg/apis/camel/v1alpha1/knative"
	knativeutil "github.com/apache/camel-k/pkg/util/knative"
)

type knativeTrait struct {
	BaseTrait       `property:",squash"`
	Configuration   string `property:"configuration"`
	ChannelSources  string `property:"channel-sources"`
	ChannelSinks    string `property:"channel-sinks"`
	EndpointSources string `property:"endpoint-sources"`
	EndpointSinks   string `property:"endpoint-sinks"`
	Auto            *bool  `property:"auto"`
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

	if t.Auto == nil || *t.Auto {
		if t.ChannelSources == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.ExtractChannelNames(meta.FromURIs)...)
				return true
			})

			t.ChannelSources = strings.Join(items, ",")
		}
		if t.EndpointSinks == "" {
			items := make([]string, 0)

			metadata.Each(e.CamelCatalog, e.Integration.Spec.Sources, func(_ int, meta metadata.IntegrationMetadata) bool {
				items = append(items, knativeutil.ExtractChannelNames(meta.ToURIs)...)
				return true
			})

			t.EndpointSinks = strings.Join(items, ",")
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
	for _, ch := range channels {
		sub := knativeutil.CreateSubscription(e.Integration.Namespace, ch, e.Integration.Name)
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
		return fmt.Errorf("cannot use the same channels as source and synk (%s)", is.List())
	}

	// Sources
	for _, ch := range sources {
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
	for _, ch := range sinks {
		if env.ContainsService(ch, knativeapi.CamelServiceTypeChannel) {
			continue
		}

		c, err := knativeutil.GetChannel(t.ctx, t.client, e.Integration.Namespace, ch)
		if err != nil {
			return err
		}
		if c == nil || c.Status.Address.Hostname == "" {
			return errors.New("cannot find address of channel " + ch)
		}

		svc := knativeapi.CamelServiceDefinition{
			Name:        ch,
			Host:        c.Status.Address.Hostname,
			Port:        80,
			Protocol:    knativeapi.CamelProtocolHTTP,
			ServiceType: knativeapi.CamelServiceTypeChannel,
			Metadata: map[string]string{
				knativeapi.CamelMetaServicePath: "/",
			},
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

		s, err := knativeutil.GetService(t.ctx, t.client, e.Integration.Namespace, endpoint)
		if err != nil {
			return err
		}
		if s == nil || s.Status.Address == nil || s.Status.Address.Hostname == "" {
			return errors.New("cannot find address of endpoint " + endpoint)
		}

		svc := knativeapi.CamelServiceDefinition{
			Name:        endpoint,
			Host:        s.Status.Address.Hostname,
			Port:        80,
			Protocol:    knativeapi.CamelProtocolHTTP,
			ServiceType: knativeapi.CamelServiceTypeEndpoint,
			Metadata: map[string]string{
				knativeapi.CamelMetaServicePath: "/",
			},
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
