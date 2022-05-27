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

package source

import (
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/stretchr/testify/assert"
)

const JavaSourceKameletEip = `
from("direct:start")
    .kamelet("foo/bar?baz=test")
`

const JavaSourceKameletEndpoint = `
from("direct:start")
    .to("kamelet:foo/bar?baz=test")
`

const JavaSourceWireTapEip = `
from("direct:start")
    .wireTap("kamelet:foo/bar?baz=test")
`

func TestJavaSourceKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   JavaSourceKameletEip,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   JavaSourceKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   JavaSourceWireTapEip,
			kamelets: []string{"foo/bar"},
		},
	}

	for i := range tc {
		test := tc[i]
		t.Run(fmt.Sprintf("TestJavaSourceKamelet-%d", i), func(t *testing.T) {
			code := v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Content: test.source,
				},
			}

			catalog, err := camel.DefaultCatalog()
			assert.Nil(t, err)

			meta := NewMetadata()
			inspector := JavaSourceInspector{
				baseInspector: baseInspector{
					catalog: catalog,
				},
			}

			err = inspector.Extract(code, &meta)
			assert.Nil(t, err)
			assert.True(t, meta.RequiredCapabilities.IsEmpty())

			for _, k := range test.kamelets {
				assert.Contains(t, meta.Kamelets, k)
			}
		})
	}
}
