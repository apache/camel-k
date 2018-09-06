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

package build

import (
	"testing"
	"context"
	"github.com/stretchr/testify/assert"
	build "github.com/apache/camel-k/pkg/build/api"
	"time"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestBuild(t *testing.T) {
	ctx := context.TODO()
	buildManager := NewBuildManager(ctx, test.GetTargetNamespace())
	identifier := build.BuildIdentifier{
		Name: "example",
		Digest: "sadsadasdsadasdafwefwef",
	}
	buildManager.Start(build.BuildSource{
		Identifier: identifier,
		Code: code(),
	})

	deadline := time.Now().Add(5 * time.Minute)
	var result build.BuildResult
	for time.Now().Before(deadline) {
		result = buildManager.Get(identifier)
		if result.Status == build.BuildStatusCompleted || result.Status == build.BuildStatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.NotEqual(t, build.BuildStatusError, result.Status)
	assert.Equal(t, build.BuildStatusCompleted, result.Status)
	assert.Regexp(t, ".*/.*/.*:.*", result.Image)
}

func TestFailedBuild(t *testing.T) {

	ctx := context.TODO()
	buildManager := NewBuildManager(ctx, test.GetTargetNamespace())
	identifier := build.BuildIdentifier{
		Name: "example",
		Digest: "545454",
	}
	buildManager.Start(build.BuildSource{
		Identifier: identifier,
		Code: code() + "XX",
	})

	deadline := time.Now().Add(5 * time.Minute)
	var result build.BuildResult
	for time.Now().Before(deadline) {
		result = buildManager.Get(identifier)
		if result.Status == build.BuildStatusCompleted || result.Status == build.BuildStatusError {
			break
		}
		time.Sleep(2 * time.Second)
	}

	assert.Equal(t, build.BuildStatusError, result.Status)
	assert.NotEqual(t, build.BuildStatusCompleted, result.Status)
	assert.Empty(t, result.Image)
}

func code() string {
	return `package kamel;

import org.apache.camel.builder.RouteBuilder;

public class Routes extends RouteBuilder {

	@Override
    public void configure() throws Exception {
        from("timer:tick")
		  .to("log:info");
    }

}
`
}