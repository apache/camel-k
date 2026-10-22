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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
)

func TestTraitConfiguration(t *testing.T) {
	cl, err := internal.NewFakeClient()
	require.NoError(t, err)
	env := Environment{
		Ctx:    context.Background(),
		Client: cl,
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Traits: v1.Traits{
					Environment: &traitv1.EnvironmentTrait{
						ContainerMeta: ptr.To(true),
						HTTPProxy:     ptr.To(false),
					},
					Service: &traitv1.ServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto: ptr.To(true),
					},
				},
			},
		},
	}
	c := NewCatalog(nil)
	require.NoError(t, c.Configure(&env))
	environment, ok := c.GetTrait("environment").(*environmentTrait)
	require.True(t, ok)
	assert.True(t, *environment.ContainerMeta)
	assert.False(t, *environment.HTTPProxy)
	service, ok := c.GetTrait("service").(*serviceTrait)
	require.True(t, ok)
	assert.True(t, *service.Enabled)
	assert.True(t, *service.Auto)
}
