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

package cmd

import (
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigureStaticTraits(t *testing.T) {
	traits := v1.Traits{}
	c := trait.NewCatalog(nil)
	options := []string{
		"camel.runtime-version=1.2.3",
		"container.port=1234",
		"container.expose=true",
		"container.image-pull-policy=Never",
		"environment.vars=V1=X",
		"environment.vars=V2=Y",
	}
	err := configureTraits(options, &traits, c)
	assert.Nil(t, err)
	assert.Equal(t, "1.2.3", traits.Camel.RuntimeVersion)
	assert.Equal(t, 1234, traits.Container.Port)
	assert.Equal(t, true, *traits.Container.Expose)
	assert.Equal(t, corev1.PullNever, traits.Container.ImagePullPolicy)
	assert.Equal(t, []string{"V1=X", "V2=Y"}, traits.Environment.Vars)
}
