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
	"github.com/apache/camel-k/pkg/build/assemble"
	"github.com/apache/camel-k/pkg/build/publish"
	"testing"
	"time"

	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/stretchr/testify/assert"
)

func TestBuildManagerBuild(t *testing.T) {
	ctx := context.TODO()
	namespace := getTargetNamespace()
	assembler := assemble.NewMavenAssembler(ctx)
	publisher := publish.NewS2IPublisher(ctx, namespace)
	buildManager := build.NewManager(ctx, assembler, publisher)
	identifier := build.Identifier{
		Name:      "man-test",
		Qualifier: digest.Random(),
	}
	buildManager.Start(build.Request{
		Identifier: identifier,
		Code: build.Source{
			Content: createTimerToLogIntegrationCode(),
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-core",
			"camel:telegram",
		},
	})

	deadline := time.Now().Add(5 * time.Minute)
	var result build.Result
	for time.Now().Before(deadline) {
		result = buildManager.Get(identifier)
		if result.Status == build.StatusCompleted || result.Status == build.StatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.NotEqual(t, build.StatusError, result.Status)
	assert.Equal(t, build.StatusCompleted, result.Status)
	assert.Regexp(t, ".*/.*/.*:.*", result.Image)
}

func TestBuildManagerFailedBuild(t *testing.T) {

	ctx := context.TODO()
	namespace := getTargetNamespace()
	assembler := assemble.NewMavenAssembler(ctx)
	publisher := publish.NewS2IPublisher(ctx, namespace)
	buildManager := build.NewManager(ctx, assembler, publisher)
	identifier := build.Identifier{
		Name:      "man-test-2",
		Qualifier: digest.Random(),
	}
	buildManager.Start(build.Request{
		Identifier: identifier,
		Code: build.Source{
			Content: createTimerToLogIntegrationCode(),
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-cippalippa",
		},
	})

	deadline := time.Now().Add(5 * time.Minute)
	var result build.Result
	for time.Now().Before(deadline) {
		result = buildManager.Get(identifier)
		if result.Status == build.StatusCompleted || result.Status == build.StatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.Equal(t, build.StatusError, result.Status)
	assert.NotEqual(t, build.StatusCompleted, result.Status)
	assert.Empty(t, result.Image)
}
