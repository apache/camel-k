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

package images

import (
	"fmt"
	"github.com/apache/camel-k/pkg/util/camel"
	"strings"

	"github.com/apache/camel-k/version"
)

// BaseRepository is the docker repository that contains images
const (
	BaseRepository = "camelk"
	ImagePrefix    = "camel-base-knative-"
)

// BaseDependency is a required dependency that must be found in the list
var BaseDependency = "camel-k:knative"

// StandardDependencies are common dependencies included in the image
var StandardDependencies = map[string]bool{
	"camel:core":   true,
	"runtime:jvm":  true,
	"runtime:yaml": true,
}

// LookupPredefinedImage is used to find a suitable predefined image if available
func LookupPredefinedImage(dependencies []string) string {

	realDependencies := make([]string, 0)
	baseDependencyFound := false
	for _, d := range dependencies {
		if _, std := StandardDependencies[d]; std {
			continue
		}
		if d == BaseDependency {
			baseDependencyFound = true
			continue
		}
		realDependencies = append(realDependencies, d)
	}

	if !baseDependencyFound {
		return ""
	}
	if len(realDependencies) == 0 {
		return PredefinedImageNameFor("core")
	}
	if len(realDependencies) != 1 {
		return ""
	}

	otherDep := realDependencies[0]
	camelPrefix := "camel:"
	if !strings.HasPrefix(otherDep, camelPrefix) {
		return ""
	}
	comp := strings.TrimPrefix(otherDep, camelPrefix)
	if !isInCamelCatalog(comp) {
		return ""
	}
	return PredefinedImageNameFor(comp)
}

// PredefinedImageNameFor --
func PredefinedImageNameFor(comp string) string {
	return fmt.Sprintf("%s/%s%s:%s", BaseRepository, ImagePrefix, comp, version.Version)
}

func isInCamelCatalog(comp string) bool {
	if _, ok := camel.Runtime.Artifacts["camel-" + comp]; ok {
		return true
	}
	return false
}
