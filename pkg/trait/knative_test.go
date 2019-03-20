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
	"context"
	"testing"

	"github.com/apache/camel-k/pkg/util/envvar"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/test"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/scylladb/go-set/strset"

	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/kubernetes"

	knativeapi "github.com/apache/camel-k/pkg/apis/camel/v1alpha1/knative"
	k8sutils "github.com/apache/camel-k/pkg/util/kubernetes"
	eventing "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKnativeEnvConfiguration(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(context.TODO(), nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{
				Profile:   v1alpha1.TraitProfileKnative,
				Sources:   []v1alpha1.SourceSpec{},
				Resources: []v1alpha1.ResourceSpec{},
				Traits: map[string]v1alpha1.IntegrationTraitSpec{
					"knative": {
						Configuration: map[string]string{
							"enabled":          "true",
							"auto":             "false",
							"channel-sources":  "channel-source-1",
							"channel-sinks":    "channel-sink-1",
							"endpoint-sources": "endpoint-source-1",
							"endpoint-sinks":   "endpoint-sink-1,endpoint-sink-2",
						},
					},
				},
			},
		},
		IntegrationContext: &v1alpha1.IntegrationContext{
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1alpha1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1alpha1.IntegrationPlatformRegistrySpec{Address: "registry"},
				},
				Profile: v1alpha1.TraitProfileKnative,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      k8sutils.NewCollection(),
		Classpath:      strset.New(),
	}

	c, err := NewFakeClient("ns")
	assert.Nil(t, err)

	client := &FakeClient{Client: c, Interface: nil}

	tc := NewCatalog(context.TODO(), client)

	err = tc.configure(&environment)
	assert.Nil(t, err)

	tr := tc.GetTrait("knative").(*knativeTrait)

	ok, err := tr.Configure(&environment)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = tr.Apply(&environment)
	assert.Nil(t, err)

	kc := envvar.Get(environment.EnvVars, "CAMEL_KNATIVE_CONFIGURATION")
	assert.NotNil(t, kc)

	ne := knativeapi.NewCamelEnvironment()
	err = ne.Deserialize(kc.Value)
	assert.Nil(t, err)

	cSource1 := ne.FindService("channel-source-1", knativeapi.CamelServiceTypeChannel)
	assert.NotNil(t, cSource1)
	assert.Equal(t, "0.0.0.0", cSource1.Host)

	cSink1 := ne.FindService("channel-sink-1", knativeapi.CamelServiceTypeChannel)
	assert.NotNil(t, cSink1)
	assert.Equal(t, "channel-sink-1.host", cSink1.Host)

	eSource1 := ne.FindService("endpoint-source-1", knativeapi.CamelServiceTypeEndpoint)
	assert.NotNil(t, eSource1)
	assert.Equal(t, "0.0.0.0", eSource1.Host)

	eSink1 := ne.FindService("endpoint-sink-1", knativeapi.CamelServiceTypeEndpoint)
	assert.NotNil(t, eSink1)
	assert.Equal(t, "endpoint-sink-1.host", eSink1.Host)
	eSink2 := ne.FindService("endpoint-sink-2", knativeapi.CamelServiceTypeEndpoint)
	assert.NotNil(t, eSink2)
	assert.Equal(t, "endpoint-sink-2.host", eSink2.Host)

}

type FakeClient struct {
	controller.Client
	kubernetes.Interface
}

func (c *FakeClient) GetScheme() *runtime.Scheme {
	return nil
}

func NewFakeClient(namespace string) (controller.Client, error) {
	scheme := clientscheme.Scheme

	// Setup Scheme for all resources
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return fake.NewFakeClientWithScheme(
		scheme,
		&eventing.Channel{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Channel",
				APIVersion: eventing.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "channel-sink-1",
			},
			Status: eventing.ChannelStatus{
				Address: duckv1alpha1.Addressable{
					Hostname: "channel-sink-1.host",
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
				Address: &duckv1alpha1.Addressable{
					Hostname: "endpoint-sink-1.host",
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
				Address: &duckv1alpha1.Addressable{
					Hostname: "endpoint-sink-2.host",
				},
			},
		},
	), nil
}
