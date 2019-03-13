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
	"fmt"
	"testing"
	"time"

	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/s2i"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func handler(in chan builder.Result, out chan builder.Result) {
	for {
		select {
		case res := <-in:
			if res.Status == builder.StatusCompleted || res.Status == builder.StatusError {
				out <- res
				return
			}
		case <-time.After(5 * time.Minute):
			fmt.Println("timeout 1")
			close(out)
			return
		}
	}
}

func TestBuildManagerBuild(t *testing.T) {
	namespace := getTargetNamespace()
	b := builder.New(testClient, namespace)

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	r := builder.Request{
		C:              cancellable.NewContext(),
		Catalog:        catalog,
		RuntimeVersion: defaults.RuntimeVersion,
		Meta: metav1.ObjectMeta{
			Name:            "man-test",
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
		Steps: s2i.DefaultSteps,
	}

	hc := make(chan builder.Result)
	rc := make(chan builder.Result)

	go func() {
		handler(hc, rc)
	}()
	go func() {
		b.Submit(r, func(res *builder.Result) {
			hc <- *res
		})
	}()

	result, ok := <-rc
	assert.True(t, ok)
	assert.NotEqual(t, builder.StatusError, result.Status)
	assert.Equal(t, builder.StatusCompleted, result.Status)
	assert.Regexp(t, ".*/.*/.*:.*", result.Image)
}

func TestBuildManagerFailedBuild(t *testing.T) {
	namespace := getTargetNamespace()
	b := builder.New(testClient, namespace)

	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	r := builder.Request{
		C:              cancellable.NewContext(),
		Catalog:        catalog,
		RuntimeVersion: defaults.RuntimeVersion,
		Meta: metav1.ObjectMeta{
			Name:            "man-test",
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
		Steps: s2i.DefaultSteps,
	}

	hc := make(chan builder.Result)
	rc := make(chan builder.Result)

	go func() {
		handler(hc, rc)
	}()
	go func() {
		b.Submit(r, func(res *builder.Result) {
			hc <- *res
		})
	}()

	result, ok := <-rc
	assert.True(t, ok)
	assert.Equal(t, builder.StatusError, result.Status)
	assert.NotEqual(t, builder.StatusCompleted, result.Status)
}
