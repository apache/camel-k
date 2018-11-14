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
	"github.com/apache/camel-k/pkg/builder/s2i"
	"github.com/apache/camel-k/pkg/util/digest"

	"github.com/stretchr/testify/assert"
)

func TestBuildManagerBuild(t *testing.T) {
	ctx := context.TODO()
	namespace := getTargetNamespace()
	b := builder.New(ctx, namespace)

	r := builder.Request{
		Identifier: builder.Identifier{
			Name:      "man-test",
			Qualifier: digest.Random(),
		},
		Code: v1alpha1.SourceSpec{
			Content: createTimerToLogIntegrationCode(),
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-core",
			"camel:telegram",
		},
		Steps: s2i.DefaultSteps,
	}

	b.Submit(r)

	deadline := time.Now().Add(5 * time.Minute)
	var result builder.Result

	for time.Now().Before(deadline) {
		result = b.Submit(r)
		if result.Status == builder.StatusCompleted || result.Status == builder.StatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.NotEqual(t, builder.StatusError, result.Status)
	assert.Equal(t, builder.StatusCompleted, result.Status)
	assert.Regexp(t, ".*/.*/.*:.*", result.Image)
}

func TestBuildManagerFailedBuild(t *testing.T) {
	ctx := context.TODO()
	namespace := getTargetNamespace()
	b := builder.New(ctx, namespace)

	r := builder.Request{
		Identifier: builder.Identifier{
			Name:      "man-test",
			Qualifier: digest.Random(),
		},
		Code: v1alpha1.SourceSpec{
			Content: createTimerToLogIntegrationCode(),
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-cippalippa",
		},
		Steps: s2i.DefaultSteps,
	}

	b.Submit(r)

	deadline := time.Now().Add(5 * time.Minute)
	var result builder.Result
	for time.Now().Before(deadline) {
		result = b.Submit(r)
		if result.Status == builder.StatusCompleted || result.Status == builder.StatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.Equal(t, builder.StatusError, result.Status)
	assert.NotEqual(t, builder.StatusCompleted, result.Status)
	assert.Empty(t, result.Image)
}
