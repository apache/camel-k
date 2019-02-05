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
	"strconv"
	"testing"

	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"

	"github.com/apache/camel-k/version"
	"github.com/stretchr/testify/assert"
)

func TestImageLookup(t *testing.T) {
	cases := []struct {
		dependencies []string
		image        string
	}{
		{
			dependencies: []string{"camel:telegram"},
		},
		{
			dependencies: []string{"camel:telegram", "camel:core"},
		},
		{
			dependencies: []string{"camel:telegram", "camel:core", "camel-k:knative"},
			image:        BaseRepository + "/" + ImagePrefix + "telegram:" + version.Version,
		},
		{
			dependencies: []string{"camel:telegram", "camel-k:knative"},
			image:        BaseRepository + "/" + ImagePrefix + "telegram:" + version.Version,
		},
		{
			dependencies: []string{"camel:telegram", "camel:core", "camel-k:knative", "camel:dropbox"},
		},
		{
			dependencies: []string{"camel:core", "camel-k:knative"},
			image:        BaseRepository + "/" + ImagePrefix + "core:" + version.Version,
		},
		{
			dependencies: []string{"camel:dropbox", "camel:core", "camel-k:knative", "runtime:jvm"},
			image:        BaseRepository + "/" + ImagePrefix + "dropbox:" + version.Version,
		},
		{
			dependencies: []string{"camel:dropbox", "camel:core", "camel-k:knative", "runtime:jvm", "runtime:yaml"},
			image:        BaseRepository + "/" + ImagePrefix + "dropbox:" + version.Version,
		},
		{
			dependencies: []string{"camel:dropbox", "camel:core", "runtime:jvm", "runtime:yaml"},
		},
		{
			dependencies: []string{"camel:dropbox", "camel:core", "camel-k:knative", "runtime:jvm", "runtime:groovy"},
		},
		{
			dependencies: []string{"camel:cippalippa", "camel:core", "camel-k:knative"},
		},
	}

	for i, tc := range cases {
		testcase := tc
		catalog := camel.Catalog(defaults.CamelVersion)

		t.Run("case-"+strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, testcase.image, LookupPredefinedImage(catalog, testcase.dependencies))
		})
	}

}
