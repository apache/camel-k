// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package test

import (
	"context"
	"testing"
	"time"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	_ "github.com/apache/camel-k/pkg/builder/kaniko"
	"github.com/apache/camel-k/pkg/builder/s2i"
	_ "github.com/apache/camel-k/pkg/builder/springboot"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/test"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestBuildManagerBuild(t *testing.T) {
	b := builder.New(testClient)

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	err = testClient.Create(context.TODO(), &v1alpha1.CamelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "catalog-test",
			Namespace: getTargetNamespace(),
		},
		Spec: catalog.CamelCatalogSpec,
	})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		assert.Error(t, err)
	}

	r := v1alpha1.BuildSpec{
		RuntimeVersion: defaults.RuntimeVersion,
		Meta: metav1.ObjectMeta{
			Name:            "man-test",
			Namespace:       getTargetNamespace(),
			ResourceVersion: "1",
		},
		Platform: v1alpha1.IntegrationPlatformSpec{
			Build: v1alpha1.IntegrationPlatformBuildSpec{
				CamelVersion:   catalog.Version,
				RuntimeVersion: defaults.RuntimeVersion,
				BaseImage:      "docker.io/fabric8/s2i-java:3.0-java8",
				Timeout: metav1.Duration{
					Duration: 5 * time.Minute,
				},
			},
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-core",
			"camel:telegram",
		},
		Steps: builder.StepIDsFor(s2i.DefaultSteps...),
	}

	result := b.Build(r)

	assert.NotEqual(t, v1alpha1.BuildPhaseFailed, result.Phase)
	assert.Equal(t, v1alpha1.BuildPhaseSucceeded, result.Phase)
	assert.Regexp(t, ".*/.*/.*:.*", result.Image)
}

func TestBuildManagerFailedBuild(t *testing.T) {
	b := builder.New(testClient)

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	err = testClient.Create(context.TODO(), &v1alpha1.CamelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "catalog-test",
			Namespace: getTargetNamespace(),
		},
		Spec: catalog.CamelCatalogSpec,
	})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		assert.Error(t, err)
	}

	r := v1alpha1.BuildSpec{
		RuntimeVersion: defaults.RuntimeVersion,
		Meta: metav1.ObjectMeta{
			Name:            "man-test",
			Namespace:       getTargetNamespace(),
			ResourceVersion: "1",
		},
		Platform: v1alpha1.IntegrationPlatformSpec{
			Build: v1alpha1.IntegrationPlatformBuildSpec{
				CamelVersion:   catalog.Version,
				RuntimeVersion: defaults.RuntimeVersion,
				BaseImage:      "docker.io/fabric8/s2i-java:3.0-java8",
				Timeout: metav1.Duration{
					Duration: 5 * time.Minute,
				},
			},
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-cippalippa",
		},
		Steps: builder.StepIDsFor(s2i.DefaultSteps...),
	}

	result := b.Build(r)

	assert.Equal(t, v1alpha1.BuildPhaseFailed, result.Phase)
	assert.NotEqual(t, v1alpha1.BuildPhaseSucceeded, result.Phase)
}
