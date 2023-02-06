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

// DefaultKits --.
var DefaultKits = []string{
	"platform-integration-kit-jvm.yaml",
	"platform-integration-kit-groovy.yaml",
	"platform-integration-kit-kotlin.yaml",
}

// KnativeKits --.
var KnativeKits = []string{
	"platform-integration-kit-knative.yaml",
}

// GetKits --.
func GetKits() []string {
	return append(DefaultKits, KnativeKits...)
}

// GetKitsNames --.
func GetKitsNames() []string {
	ctxs := GetKits()
	names := make([]string, 0, len(ctxs))

	for _, r := range ctxs {
		r = strings.TrimPrefix(r, "platform-integration-kit-")
		r = strings.TrimSuffix(r, ".yaml")

		names = append(names, r)
	}

	return names
}
