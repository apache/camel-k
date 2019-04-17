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

package springboot

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
)

func initialize(ctx *builder.Context) error {
	// do not take into account any image that does not have spring-boot
	// as required dependency to avoid picking up a base image with wrong
	// classpath or layout
	ctx.ContextFilter = func(context *v1alpha1.IntegrationContext) bool {
		for _, i := range context.Spec.Dependencies {
			if i == "runtime:spring" {
				return true
			}
		}

		return false
	}

	return nil
}
