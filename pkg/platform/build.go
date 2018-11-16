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

package platform

import (
	"context"
	"errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/kaniko"
	"github.com/apache/camel-k/pkg/builder/s2i"
)

// gBuilder is the current builder
// Note: it cannot be changed at runtime, needs a operator restart
var gBuilder builder.Builder

// GetPlatformBuilder --
func GetPlatformBuilder(ctx context.Context, namespace string) (builder.Builder, error) {
	if gBuilder != nil {
		return gBuilder, nil
	}

	gBuilder = builder.New(ctx, namespace)

	return gBuilder, nil
}

// NewBuildRequest --
func NewBuildRequest(ctx context.Context, context *v1alpha1.IntegrationContext) (builder.Request, error) {
	req := builder.Request{
		Identifier: builder.Identifier{
			Name:      "context-" + context.Name,
			Qualifier: context.ResourceVersion,
		},
		Dependencies: context.Spec.Dependencies,
		Steps:        kaniko.DefaultSteps,
	}

	p, err := GetCurrentPlatform(context.Namespace)
	if err != nil {
		return req, err
	}

	req.Platform = p.Spec

	if SupportsS2iPublishStrategy(p) {
		req.Steps = s2i.DefaultSteps
	} else if SupportsKanikoPublishStrategy(p) {
		req.Steps = kaniko.DefaultSteps
		req.BuildDir = kaniko.BuildDir
	} else {
		return req, errors.New("unsupported platform configuration")
	}

	return req, nil
}
