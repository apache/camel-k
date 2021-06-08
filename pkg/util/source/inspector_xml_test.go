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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/camel-k/pkg/util/camel"
)

const XMLKameletEip = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <kamelet name="foo/bar?baz=test"/>
    <to uri="mock:result"/>
  </route>
</camelContext>
`
const XMLKameletEndpoint = `
<camelContext xmlns="http://camel.apache.org/schema/spring">
  <route>
    <from uri="direct:start"/>
    <to uri="kamelet:foo/bar?baz=test"/>
  </route>
</camelContext>
`

func TestXMLKamelet(t *testing.T) {
	tc := []struct {
		source   string
		kamelets []string
	}{
		{
			source:   XMLKameletEip,
			kamelets: []string{"foo/bar"},
		},
		{
			source:   XMLKameletEndpoint,
			kamelets: []string{"foo/bar"},
		},
	}

	for i, test := range tc {
		t.Run(fmt.Sprintf("TestXMLKamelet-%d", i), func(t *testing.T) {
			code := v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Content: test.source,
				},
			}

			catalog, err := camel.DefaultCatalog()
			assert.Nil(t, err)

			meta := NewMetadata()
			inspector := XMLInspector{
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
