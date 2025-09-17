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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1"
	messaging "knative.dev/eventing/pkg/apis/messaging/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	knativeapi "github.com/apache/camel-k/v2/pkg/apis/camel/v1/knative"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/knative"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	k8sutils "github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestKnativeEnvConfigurationFromTrait(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto:            ptr.To(false),
						ChannelSources:  []string{"channel-source-1"},
						ChannelSinks:    []string{"channel-sink-1"},
						EndpointSources: []string{"endpoint-source-1"},
						EndpointSinks:   []string{"endpoint-sink-1", "endpoint-sink-2"},
						EventSources:    []string{"knative:event"},
						EventSinks:      []string{"knative:event"},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
				Profile: v1.TraitProfileKnative,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	tc := NewCatalog(c)

	err = tc.Configure(&environment)
	require.NoError(t, err)

	tr, _ := tc.GetTrait("knative").(*knativeTrait)
	ok, condition, err := tr.Configure(&environment)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = tr.Apply(&environment)
	require.NoError(t, err)

	ne, err := fromCamelProperties(environment.ApplicationProperties)
	require.NoError(t, err)

	cSource1 := ne.FindService("channel-source-1", knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeChannel, "messaging.knative.dev/v1", "Channel")
	assert.NotNil(t, cSource1)
	assert.Empty(t, cSource1.URL)

	cSink1 := ne.FindService("channel-sink-1", knativeapi.CamelEndpointKindSink, knativeapi.CamelServiceTypeChannel, "messaging.knative.dev/v1", "Channel")
	assert.NotNil(t, cSink1)
	assert.Equal(t, "http://channel-sink-1.host/", cSink1.URL)

	eSource1 := ne.FindService("endpoint-source-1", knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEndpoint, "serving.knative.dev/v1", "Service")
	assert.NotNil(t, eSource1)
	assert.Empty(t, eSource1.URL)

	eSink1 := ne.FindService("endpoint-sink-1", knativeapi.CamelEndpointKindSink, knativeapi.CamelServiceTypeEndpoint, "serving.knative.dev/v1", "Service")
	assert.NotNil(t, eSink1)
	assert.Equal(t, "http://endpoint-sink-1.host/", eSink1.URL)
	eSink2 := ne.FindService("endpoint-sink-2", knativeapi.CamelEndpointKindSink, knativeapi.CamelServiceTypeEndpoint, "serving.knative.dev/v1", "Service")
	assert.NotNil(t, eSink2)
	assert.Equal(t, "http://endpoint-sink-2.host/", eSink2.URL)

	eEventSource := ne.FindService("default", knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEvent, "eventing.knative.dev/v1", "Broker")
	assert.NotNil(t, eEventSource)
	eEventSink := ne.FindService("default", knativeapi.CamelEndpointKindSink, knativeapi.CamelServiceTypeEvent, "eventing.knative.dev/v1", "Broker")
	assert.NotNil(t, eEventSink)
	assert.Equal(t, "http://broker-default.host/", eEventSink.URL)

	assert.NotNil(t, environment.Resources.GetKnativeSubscription(func(subscription *messaging.Subscription) bool {
		return assert.Equal(t, "channel-source-1-test", subscription.Name)
	}))

	assert.NotNil(t, environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return assert.Equal(t, "default-test", trigger.Name)
	}))
}

func TestKnativeEnvConfigurationFromSource(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:endpoint/s3fileMover1")
											.log("${body}");

										from("knative:channel/channel-source-1")
 											.log("${body}");

										from("knative:event/evt.type")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
				Profile: v1.TraitProfileKnative,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	tc := NewCatalog(c)

	err = tc.Configure(&environment)
	require.NoError(t, err)

	tr, _ := tc.GetTrait("knative").(*knativeTrait)

	ok, condition, err := tr.Configure(&environment)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	err = tr.Apply(&environment)
	require.NoError(t, err)

	ne, err := fromCamelProperties(environment.ApplicationProperties)
	require.NoError(t, err)

	source := ne.FindService("s3fileMover1", knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEndpoint, "serving.knative.dev/v1", "Service")
	assert.NotNil(t, source)
	assert.Empty(t, source.URL)
	assert.Empty(t, source.Metadata[knativeapi.CamelMetaKnativeReply])

	channel := ne.FindService("channel-source-1", knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeChannel, "", "")
	assert.NotNil(t, channel)
	assert.Equal(t, boolean.FalseString, channel.Metadata[knativeapi.CamelMetaKnativeReply])

	broker := ne.FindService("evt.type", knativeapi.CamelEndpointKindSource, knativeapi.CamelServiceTypeEvent, "", "")
	assert.NotNil(t, broker)
	assert.Equal(t, boolean.FalseString, broker.Metadata[knativeapi.CamelMetaKnativeReply])

	assert.NotNil(t, environment.Resources.GetKnativeSubscription(func(subscription *messaging.Subscription) bool {
		return assert.Equal(t, "channel-source-1-test", subscription.Name)
	}))

	assert.NotNil(t, environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return assert.Equal(t, "default-test-evttype", trigger.Name)
	}))
}

func TestKnativeTriggerExplicitFilterConfig(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event/evt.type")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Filters: []string{"source=my-source"},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	trigger := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type")
	})

	assert.NotNil(t, trigger)

	assert.Equal(t, "default", trigger.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type", trigger.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype", trigger.Name)

	assert.NotNil(t, trigger.Spec.Filter)
	assert.Len(t, trigger.Spec.Filter.Attributes, 2)
	assert.Equal(t, trigger.Spec.Filter.Attributes["type"], "evt.type")
	assert.Equal(t, trigger.Spec.Filter.Attributes["source"], "my-source")
}

func TestKnativeTriggerExplicitFilterConfigNoEventTypeFilter(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event/evt.type")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Filters:         []string{"source=my-source"},
						FilterEventType: ptr.To(false),
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	trigger := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type")
	})

	assert.NotNil(t, trigger)

	assert.Equal(t, "default", trigger.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type", trigger.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype", trigger.Name)

	assert.NotNil(t, trigger.Spec.Filter)
	assert.Len(t, trigger.Spec.Filter.Attributes, 1)
	assert.Equal(t, trigger.Spec.Filter.Attributes["source"], "my-source")
}

func TestKnativeTriggerDefaultEventTypeFilter(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event/evt.type")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	trigger := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type")
	})

	assert.NotNil(t, trigger)
	assert.Equal(t, "default", trigger.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type", trigger.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype", trigger.Name)

	assert.NotNil(t, trigger.Spec.Filter)
	assert.Len(t, trigger.Spec.Filter.Attributes, 1)
	assert.Equal(t, "evt.type", trigger.Spec.Filter.Attributes["type"])
}

func TestKnativeTriggerDefaultEventTypeFilterDisabled(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event/evt.type")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						FilterEventType: ptr.To(false),
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	trigger := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type")
	})

	assert.NotNil(t, trigger)
	assert.Equal(t, "default", trigger.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type", trigger.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype", trigger.Name)

	assert.Nil(t, trigger.Spec.Filter)
}

func TestKnativeMultipleTrigger(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event/evt.type.1")
 											.log("${body}");

										from("knative:event/evt.type.2")
 											.log("${body}");

										from("knative:event")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	triggerNames := make([]string, 0)
	environment.Resources.VisitKnativeTrigger(func(trigger *eventing.Trigger) {
		triggerNames = append(triggerNames, trigger.Name)
	})

	assert.Len(t, triggerNames, 3)

	trigger1 := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type.1")
	})

	assert.NotNil(t, trigger1)
	assert.Equal(t, "default", trigger1.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger1.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger1.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type.1", trigger1.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype1", trigger1.Name)

	assert.NotNil(t, trigger1.Spec.Filter)
	assert.Len(t, trigger1.Spec.Filter.Attributes, 1)
	assert.Equal(t, "evt.type.1", trigger1.Spec.Filter.Attributes["type"])

	trigger2 := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type.2")
	})

	assert.NotNil(t, trigger2)
	assert.Equal(t, "default", trigger2.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger2.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger2.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type.2", trigger2.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype2", trigger2.Name)

	assert.NotNil(t, trigger2.Spec.Filter)
	assert.Len(t, trigger2.Spec.Filter.Attributes, 1)
	assert.Equal(t, "evt.type.2", trigger2.Spec.Filter.Attributes["type"])

	trigger3 := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "")
	})

	assert.NotNil(t, trigger3)
	assert.Equal(t, "default", trigger3.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger3.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger3.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/", trigger3.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test", trigger3.Name)

	assert.Nil(t, trigger3.Spec.Filter)
}

func TestKnativeMultipleTriggerAdditionalFilterConfig(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event/evt.type.1")
 											.log("${body}");

										from("knative:event/evt.type.2")
 											.log("${body}");

										from("knative:event")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Filters: []string{"subject=Hello"},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	triggerNames := make([]string, 0)
	environment.Resources.VisitKnativeTrigger(func(trigger *eventing.Trigger) {
		triggerNames = append(triggerNames, trigger.Name)
	})

	assert.Len(t, triggerNames, 3)

	trigger1 := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type.1")
	})

	assert.NotNil(t, trigger1)
	assert.Equal(t, "default", trigger1.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger1.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger1.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type.1", trigger1.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype1", trigger1.Name)

	assert.NotNil(t, trigger1.Spec.Filter)
	assert.Len(t, trigger1.Spec.Filter.Attributes, 2)
	assert.Equal(t, "evt.type.1", trigger1.Spec.Filter.Attributes["type"])
	assert.Equal(t, "Hello", trigger1.Spec.Filter.Attributes["subject"])

	trigger2 := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type.2")
	})

	assert.NotNil(t, trigger2)
	assert.Equal(t, "default", trigger2.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger2.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger2.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type.2", trigger2.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype2", trigger2.Name)

	assert.NotNil(t, trigger2.Spec.Filter)
	assert.Len(t, trigger2.Spec.Filter.Attributes, 2)
	assert.Equal(t, "evt.type.2", trigger2.Spec.Filter.Attributes["type"])
	assert.Equal(t, "Hello", trigger2.Spec.Filter.Attributes["subject"])

	trigger3 := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "")
	})

	assert.NotNil(t, trigger3)
	assert.Equal(t, "default", trigger3.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger3.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger3.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/", trigger3.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test", trigger3.Name)

	assert.NotNil(t, trigger3.Spec.Filter)
	assert.Len(t, trigger3.Spec.Filter.Attributes, 1)
	assert.Equal(t, "Hello", trigger3.Spec.Filter.Attributes["subject"])
}

func TestKnativeTriggerNoEventType(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	trigger := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "")
	})

	assert.NotNil(t, trigger)
	assert.Equal(t, "default", trigger.Spec.Broker)
	assert.Equal(t, serving.SchemeGroupVersion.String(), trigger.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/", trigger.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test", trigger.Name)

	assert.Nil(t, trigger.Spec.Filter)
}

func TestKnativeTriggerNoServingAvailable(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	fakeClient := c.(*internal.FakeClient) //nolint
	fakeClient.DisableKnativeServing()

	traitCatalog := NewCatalog(c)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "route.java",
							Content: `
								public class CartoonMessagesMover extends RouteBuilder {
									public void configure() {
										from("knative:event/evt.type")
 											.log("${body}");
									}
								}
							`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// don't care about conditions in this unit test
	_, _, err = traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("knative"))

	trigger := environment.Resources.GetKnativeTrigger(func(trigger *eventing.Trigger) bool {
		return trigger.Name == knative.GetTriggerName("default", "test", "evt.type")
	})

	assert.NotNil(t, trigger)

	assert.Equal(t, "default", trigger.Spec.Broker)
	assert.Equal(t, "v1", trigger.Spec.Subscriber.Ref.APIVersion)
	assert.Equal(t, "Service", trigger.Spec.Subscriber.Ref.Kind)
	assert.Equal(t, "/events/evt.type", trigger.Spec.Subscriber.URI.Path)
	assert.Equal(t, "default-test-evttype", trigger.Name)

	assert.NotNil(t, trigger.Spec.Filter)
	assert.Len(t, trigger.Spec.Filter.Attributes, 1)
	assert.Equal(t, "evt.type", trigger.Spec.Filter.Attributes["type"])
}

func TestKnativePlatformHttpConfig(t *testing.T) {
	sources := []v1.SourceSpec{
		{
			DataSpec: v1.DataSpec{
				Name:    "source-endpoint.java",
				Content: `from("knative:endpoint/ep").log("${body}");`,
			},
			Language: v1.LanguageJavaSource,
		},
		{
			DataSpec: v1.DataSpec{
				Name:    "source-channel.java",
				Content: `from("knative:channel/channel-source-1").log("${body}");`,
			},
			Language: v1.LanguageJavaSource,
		},
		{
			DataSpec: v1.DataSpec{
				Name:    "source-event.java",
				Content: `from("knative:event/event-source-1").log("${body}")`,
			},
			Language: v1.LanguageJavaSource,
		},
	}

	for _, ref := range sources {
		source := ref
		t.Run(source.Name, func(t *testing.T) {
			environment := NewFakeEnvironment(t, source)

			c, err := newFakeClient("ns")
			require.NoError(t, err)

			tc := NewCatalog(c)

			err = tc.Configure(&environment)
			require.NoError(t, err)

			_, _, err = tc.apply(&environment)
			require.NoError(t, err)
			assert.Contains(t, environment.Integration.Status.Capabilities, v1.CapabilityPlatformHTTP)
		})
	}
}

func TestKnativePlatformHttpDependencies(t *testing.T) {
	sources := []v1.SourceSpec{
		{
			DataSpec: v1.DataSpec{
				Name:    "source-endpoint.java",
				Content: `from("knative:endpoint/ep").log("${body}");`,
			},
			Language: v1.LanguageJavaSource,
		},
		{
			DataSpec: v1.DataSpec{
				Name:    "source-channel.java",
				Content: `from("knative:channel/channel-source-1").log("${body}");`,
			},
			Language: v1.LanguageJavaSource,
		},
		{
			DataSpec: v1.DataSpec{
				Name:    "source-event.java",
				Content: `from("knative:event/event-source-1").log("${body}")`,
			},
			Language: v1.LanguageJavaSource,
		},
	}

	for _, ref := range sources {
		source := ref
		t.Run(source.Name, func(t *testing.T) {
			environment := NewFakeEnvironment(t, source)
			environment.Integration.Status.Phase = v1.IntegrationPhaseInitialization

			c, err := newFakeClient("ns")
			require.NoError(t, err)

			tc := NewCatalog(c)

			err = tc.Configure(&environment)
			require.NoError(t, err)

			conditions, traits, err := tc.apply(&environment)
			require.NoError(t, err)
			assert.NotEmpty(t, traits)
			assert.NotEmpty(t, conditions)
			assert.Contains(t, environment.Integration.Status.Capabilities, v1.CapabilityPlatformHTTP)
			assert.Contains(t, environment.Integration.Status.Dependencies, "mvn:org.apache.camel.quarkus:camel-quarkus-platform-http")
		})
	}
}

func TestKnativeEnabled(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "route.java",
							Content: `from("timer:foo").to("knative:channel/channel-source-1")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
				Profile: v1.TraitProfileKnative,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// configure the init trait
	init := NewInitTrait()
	ok, condition, err := init.Configure(&environment)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	// apply the init trait
	require.NoError(t, init.Apply(&environment))

	// configure the knative trait
	knTrait, _ := newKnativeTrait().(*knativeTrait)
	ok, condition, err = knTrait.Configure(&environment)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	// apply the knative trait
	require.NoError(t, knTrait.Apply(&environment))
	assert.Contains(t, environment.Integration.Status.Capabilities, v1.CapabilityKnative)
}

func TestKnativeNotEnabled(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "route.java",
							Content: `from("timer:foo").to("log:info");`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
				Profile: v1.TraitProfileKnative,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	// configure the init trait
	init := NewInitTrait()
	ok, condition, err := init.Configure(&environment)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)

	// apply the init trait
	require.NoError(t, init.Apply(&environment))

	// configure the knative trait
	knTrait, _ := newKnativeTrait().(*knativeTrait)
	ok, condition, err = knTrait.Configure(&environment)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, condition)

	assert.NotContains(t, environment.Integration.Status.Capabilities, v1.CapabilityKnative)
}

func NewFakeEnvironment(t *testing.T, source v1.SourceSpec) Environment {
	t.Helper()

	client, _ := newFakeClient("ns")
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					source,
				},
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	return environment
}

func NewFakeEnvironmentForSyntheticKit(t *testing.T) Environment {
	t.Helper()
	client, _ := newFakeClient("ns")
	traitCatalog := NewCatalog(nil)

	environment := Environment{
		Catalog: traitCatalog,
		Client:  client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Traits: v1.Traits{
					Knative: &traitv1.KnativeTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
				Profile: v1.TraitProfileKnative,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	return environment
}

func newFakeClient(namespace string) (client.Client, error) {
	channelSourceURL, err := apis.ParseURL("http://channel-source-1.host/")
	if err != nil {
		return nil, err
	}
	channelSinkURL, err := apis.ParseURL("http://channel-sink-1.host/")
	if err != nil {
		return nil, err
	}
	sink1URL, err := apis.ParseURL("http://endpoint-sink-1.host/")
	if err != nil {
		return nil, err
	}
	sink2URL, err := apis.ParseURL("http://endpoint-sink-2.host/")
	if err != nil {
		return nil, err
	}
	brokerURL, err := apis.ParseURL("http://broker-default.host/")
	if err != nil {
		return nil, err
	}

	return internal.NewFakeClient(
		&messaging.Channel{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Channel",
				APIVersion: messaging.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "channel-source-1",
			},
			Status: messaging.ChannelStatus{
				ChannelableStatus: eventingduckv1.ChannelableStatus{
					AddressStatus: duckv1.AddressStatus{
						Address: &duckv1.Addressable{
							URL: channelSourceURL,
						},
					},
				},
			},
		},
		&messaging.Channel{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Channel",
				APIVersion: messaging.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "channel-sink-1",
			},
			Status: messaging.ChannelStatus{
				ChannelableStatus: eventingduckv1.ChannelableStatus{
					AddressStatus: duckv1.AddressStatus{
						Address: &duckv1.Addressable{
							URL: channelSinkURL,
						},
					},
				},
			},
		},
		&serving.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: serving.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "endpoint-sink-1",
			},
			Status: serving.ServiceStatus{
				RouteStatusFields: serving.RouteStatusFields{
					Address: &duckv1.Addressable{
						URL: sink1URL,
					},
				},
			},
		},
		&serving.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: serving.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "endpoint-sink-2",
			},
			Status: serving.ServiceStatus{
				RouteStatusFields: serving.RouteStatusFields{
					Address: &duckv1.Addressable{
						URL: sink2URL,
					},
				},
			},
		},
		&eventing.Broker{
			TypeMeta: metav1.TypeMeta{
				APIVersion: eventing.SchemeGroupVersion.String(),
				Kind:       "Broker",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "default",
			},
			Spec: eventing.BrokerSpec{},
			Status: eventing.BrokerStatus{
				AddressStatus: duckv1.AddressStatus{
					Address: &duckv1.Addressable{
						URL: brokerURL,
					},
				},
			},
		},
		&eventing.Trigger{
			TypeMeta: metav1.TypeMeta{
				APIVersion: eventing.SchemeGroupVersion.String(),
				Kind:       "Trigger",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "event-source-1",
			},
			Spec: eventing.TriggerSpec{
				Filter: &eventing.TriggerFilter{
					Attributes: eventing.TriggerFilterAttributes{
						"type": "event-source-1",
					},
				},
				Broker: "default",
				Subscriber: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: serving.SchemeGroupVersion.String(),
						Kind:       "Service",
						Name:       "event-source-1",
					},
				},
			},
		},
	)
}

func TestKnativeSinkBinding(t *testing.T) {
	source := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:    "sink.java",
			Content: `from("timer:foo").to("knative:channel/channel-sink-1?apiVersion=messaging.knative.dev%2Fv1&kind=Channel");`,
		},
		Language: v1.LanguageJavaSource,
	}

	environment := NewFakeEnvironment(t, source)
	environment.Integration.Status.Phase = v1.IntegrationPhaseDeploying

	c, err := newFakeClient("ns")
	require.NoError(t, err)

	tc := NewCatalog(c)

	err = tc.Configure(&environment)
	require.NoError(t, err)

	_, _, err = tc.apply(&environment)
	require.NoError(t, err)
	baseProp := "camel.component.knative.environment.resources[0]"
	assert.Equal(t, "channel-sink-1", environment.ApplicationProperties[baseProp+".name"])
	assert.Equal(t, "${K_SINK}", environment.ApplicationProperties[baseProp+".url"])
	assert.Equal(t, "${K_CE_OVERRIDES}", environment.ApplicationProperties[baseProp+".ceOverrides"])
	assert.Equal(t, "channel", environment.ApplicationProperties[baseProp+".type"])
	assert.Equal(t, "Channel", environment.ApplicationProperties[baseProp+".objectKind"])
	assert.Equal(t, "messaging.knative.dev/v1", environment.ApplicationProperties[baseProp+".objectApiVersion"])
	assert.Equal(t, "sink", environment.ApplicationProperties[baseProp+".endpointKind"])
}

// fromCamelProperties deserialize from properties to environment.
func fromCamelProperties(appProps map[string]string) (*knativeapi.CamelEnvironment, error) {
	env := knativeapi.NewCamelEnvironment()
	re := regexp.MustCompile(`camel.component.knative.environment.resources\[(\d+)\].name`)
	for k, v := range appProps {
		if re.MatchString(k) {
			index := re.FindStringSubmatch(k)[1]
			baseComp := fmt.Sprintf("camel.component.knative.environment.resources[%s]", index)
			url, err := url.Parse(appProps[fmt.Sprintf("%s.url", baseComp)])
			if err != nil {
				return nil, err
			}
			svc, err := knativeapi.BuildCamelServiceDefinition(
				v,
				knativeapi.CamelEndpointKind(appProps[fmt.Sprintf("%s.endpointKind", baseComp)]),
				knativeapi.CamelServiceType(appProps[fmt.Sprintf("%s.type", baseComp)]),
				*url,
				appProps[fmt.Sprintf("%s.objectApiVersion", baseComp)],
				appProps[fmt.Sprintf("%s.objectKind", baseComp)],
			)
			if err != nil {
				return nil, err
			}
			svc.Metadata[knativeapi.CamelMetaKnativeReply] = appProps[fmt.Sprintf("%s.reply", baseComp)]
			env.Services = append(env.Services, svc)
		}
	}

	return &env, nil
}

func TestKnativeSyntheticKitDefault(t *testing.T) {
	e := NewFakeEnvironmentForSyntheticKit(t)
	knTrait, _ := newKnativeTrait().(*knativeTrait)
	ok, condition, err := knTrait.Configure(&e)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, condition)
}

func TestKnativeSyntheticKitEnabled(t *testing.T) {
	e := NewFakeEnvironmentForSyntheticKit(t)
	knTrait, _ := newKnativeTrait().(*knativeTrait)
	knTrait.Enabled = ptr.To(true)
	ok, condition, err := knTrait.Configure(&e)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)
}

func TestRunKnativeEndpointWithKnativeNotInstalled(t *testing.T) {
	environment := createEnvironmentMissingEventingCRDs()
	trait, _ := newKnativeTrait().(*knativeTrait)
	environment.Integration.Spec.Sources = []v1.SourceSpec{
		{
			DataSpec: v1.DataSpec{
				Name: "test.java",
				Content: `
				from("knative:channel/test").to("log:${body};
			`,
			},
			Language: v1.LanguageJavaSource,
		},
	}
	expectedCondition := NewIntegrationCondition(
		"Knative",
		v1.IntegrationConditionKnativeAvailable,
		corev1.ConditionFalse,
		v1.IntegrationConditionKnativeNotInstalledReason,
		"integration cannot run. Knative is not installed in the cluster",
	)
	configured, condition, err := trait.Configure(environment)
	require.Error(t, err)
	assert.Equal(t, expectedCondition, condition)
	assert.False(t, configured)
}

func TestRunNonKnativeEndpointWithKnativeNotInstalled(t *testing.T) {
	environment := createEnvironmentMissingEventingCRDs()
	trait, _ := newKnativeTrait().(*knativeTrait)
	environment.Integration.Spec.Sources = []v1.SourceSpec{
		{
			DataSpec: v1.DataSpec{
				Name: "test.java",
				Content: `
				from("platform-http://my-site").to("log:${body}");
			`,
			},
			Language: v1.LanguageJavaSource,
		},
	}

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.False(t, configured)
	conditions := environment.Integration.Status.Conditions
	assert.Empty(t, conditions)
}

func createEnvironmentMissingEventingCRDs() *Environment {
	client, _ := internal.NewFakeClient()
	// disable the knative eventing api
	fakeClient := client.(*internal.FakeClient) //nolint
	fakeClient.DisableAPIGroupDiscovery("eventing.knative.dev/v1")

	replicas := int32(3)
	catalog, _ := camel.QuarkusCatalog()

	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(nil),
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Spec: v1.IntegrationSpec{
				Replicas: &replicas,
				Traits:   v1.Traits{},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
				Profile: v1.TraitProfileKubernetes,
			},
		},
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
	environment.Platform.ResyncStatusFullConfig()

	return environment
}

func TestKnativeAutoConfiguration(t *testing.T) {
	client, _ := internal.NewFakeClient()
	replicas := int32(3)
	catalog, _ := camel.QuarkusCatalog()
	environment := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(nil),
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Spec: v1.IntegrationSpec{
				Replicas: &replicas,
				Traits:   v1.Traits{},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
				Profile: v1.TraitProfileKubernetes,
			},
		},
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
	environment.Platform.ResyncStatusFullConfig()

	trait, _ := newKnativeTrait().(*knativeTrait)
	environment.Integration.Spec.Sources = []v1.SourceSpec{
		{
			DataSpec: v1.DataSpec{
				Name: "test.java",
				Content: `
				from("knative:channel/test").to("log:${body};
			`,
			},
			Language: v1.LanguageJavaSource,
		},
	}

	configured, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.Nil(t, condition)
	assert.True(t, configured)
	err = trait.Apply(environment)
	require.NoError(t, err)
	expectedTrait, _ := newKnativeTrait().(*knativeTrait)
	expectedTrait.Enabled = ptr.To(true)
	expectedTrait.SinkBinding = ptr.To(false)
	expectedTrait.ChannelSources = []string{"knative:channel/test"}
	assert.Equal(t, expectedTrait, trait)
}
