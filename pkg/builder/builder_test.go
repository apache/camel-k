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

package builder

import (
	"errors"
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestFailure(t *testing.T) {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	c, err := test.NewFakeClient()
	assert.Nil(t, err)

	b := New(c, []Step{
		NewStep("step1", InitPhase, func(i *Context) error {
			return nil
		}),
		NewStep("step2", ApplicationPublishPhase, func(i *Context) error {
			return errors.New("an error")
		}),
	})

	r := v1alpha1.BuildSpec{
		RuntimeVersion: defaults.RuntimeVersion,
		Platform: v1alpha1.IntegrationPlatformSpec{
			Build: v1alpha1.IntegrationPlatformBuildSpec{
				CamelVersion: catalog.Version,
			},
		},
	}

	result := b.Build(r)

	assert.NotNil(t, result)
	assert.Equal(t, v1alpha1.BuildPhaseFailed, result.Phase)
}
