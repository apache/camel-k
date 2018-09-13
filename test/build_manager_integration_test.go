// +build integration

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

	buildapi "github.com/apache/camel-k/pkg/build/api"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/stretchr/testify/assert"
	"github.com/apache/camel-k/pkg/build"
)

func TestBuildManagerBuild(t *testing.T) {
	ctx := context.TODO()
	buildManager := build.NewManager(ctx, GetTargetNamespace())
	identifier := buildapi.BuildIdentifier{
		Name:   "man-test",
		Digest: digest.Random(),
	}
	buildManager.Start(buildapi.BuildSource{
		Identifier: identifier,
		Code: buildapi.Code{
			Content: TimerToLogIntegrationCode(),
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-core",
			"camel:telegram",
		},
	})

	deadline := time.Now().Add(5 * time.Minute)
	var result buildapi.BuildResult
	for time.Now().Before(deadline) {
		result = buildManager.Get(identifier)
		if result.Status == buildapi.BuildStatusCompleted || result.Status == buildapi.BuildStatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.NotEqual(t, buildapi.BuildStatusError, result.Status)
	assert.Equal(t, buildapi.BuildStatusCompleted, result.Status)
	assert.Regexp(t, ".*/.*/.*:.*", result.Image)
}

func TestBuildManagerFailedBuild(t *testing.T) {

	ctx := context.TODO()
	buildManager := build.NewManager(ctx, GetTargetNamespace())
	identifier := buildapi.BuildIdentifier{
		Name:   "man-test-2",
		Digest: digest.Random(),
	}
	buildManager.Start(buildapi.BuildSource{
		Identifier: identifier,
		Code: buildapi.Code{
			Content: TimerToLogIntegrationCode(),
		},
		Dependencies: []string{
			"mvn:org.apache.camel/camel-cippalippa",
		},
	})

	deadline := time.Now().Add(5 * time.Minute)
	var result buildapi.BuildResult
	for time.Now().Before(deadline) {
		result = buildManager.Get(identifier)
		if result.Status == buildapi.BuildStatusCompleted || result.Status == buildapi.BuildStatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.Equal(t, buildapi.BuildStatusError, result.Status)
	assert.NotEqual(t, buildapi.BuildStatusCompleted, result.Status)
	assert.Empty(t, result.Image)
}
