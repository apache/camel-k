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

import "strings"

const (
	// DefaultCamelVersion --
	DefaultCamelVersion = "2.23.1"

	// DefaultBaseImage --
	DefaultBaseImage = "fabric8/s2i-java:3.0-java8"

	// DefaultLocalRepository --
	DefaultLocalRepository = "/tmp/artifacts/m2"
)

// DefaultContexts --
var DefaultContexts = []string{
	"platform-integration-context-jvm.yaml",
	"platform-integration-context-groovy.yaml",
	"platform-integration-context-kotlin.yaml",
	"platform-integration-context-spring-boot.yaml",
}

// KnativeContexts --
var KnativeContexts = []string{
	"platform-integration-context-knative.yaml",
}

// NoContext is a placeholder for a not-present context
const NoContext = "none"

// GetContexts --
func GetContexts() []string {
	return append(DefaultContexts, KnativeContexts...)
}

// GetContextsNames --
func GetContextsNames() []string {
	ctxs := GetContexts()
	names := make([]string, 0, len(ctxs))

	for _, r := range ctxs {
		r = strings.TrimPrefix(r, "platform-integration-context-")
		r = strings.TrimSuffix(r, ".yaml")

		names = append(names, r)
	}

	return names
}
