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

package trait

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"sort"
)

type dependenciesTrait struct {
	BaseTrait `property:",squash"`
}

func newDependenciesTrait() *dependenciesTrait {
	return &dependenciesTrait{
		BaseTrait: newBaseTrait("dependencies"),
	}
}

func (d *dependenciesTrait) beforeInit(environment *environment, integration *v1alpha1.Integration) error {
	meta := metadata.Extract(integration.Spec.Source)

	if meta.Language == v1alpha1.LanguageGroovy {
		util.StringSliceUniqueAdd(&integration.Spec.Dependencies, "runtime:groovy")
	} else if meta.Language == v1alpha1.LanguageKotlin {
		util.StringSliceUniqueAdd(&integration.Spec.Dependencies, "runtime:kotlin")
	}

	// jvm runtime and camel-core required by default
	util.StringSliceUniqueAdd(&integration.Spec.Dependencies, "runtime:jvm")
	util.StringSliceUniqueAdd(&integration.Spec.Dependencies, "camel:core")

	integration.Spec.Dependencies = d.mergeDependencies(integration.Spec.Dependencies, meta.Dependencies)
	// sort the dependencies to get always the same list if they don't change
	sort.Strings(integration.Spec.Dependencies)
	return nil
}

func (d *dependenciesTrait) mergeDependencies(list1 []string, list2 []string) []string {
	set := make(map[string]bool, 0)
	for _, d := range list1 {
		set[d] = true
	}
	for _, d := range list2 {
		set[d] = true
	}
	ret := make([]string, 0, len(set))
	for d := range set {
		ret = append(ret, d)
	}
	return ret
}
