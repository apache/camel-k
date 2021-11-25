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

package integrationplatform

import (
	"context"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/resources"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreate(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift

	c, err := test.NewFakeClient(&ip)
	assert.Nil(t, err)

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	assert.Nil(t, err)

	h := NewCreateAction()
	h.InjectLogger(log.Log)
	h.InjectClient(c)

	answer, err := h.Handle(context.TODO(), &ip)
	assert.Nil(t, err)
	assert.NotNil(t, answer)

	list := v1.NewCamelCatalogList()
	err = c.List(context.TODO(), &list, k8sclient.InNamespace(ip.Namespace))

	assert.Nil(t, err)
	assert.NotEmpty(t, list.Items)

	items, err := resources.WithPrefix("/camel-catelog-")
	assert.Nil(t, err)

	for _, k := range items {
		found := false

		for _, c := range list.Items {
			n := strings.TrimSuffix(k, ".yaml")
			n = strings.ToLower(n)

			if c.Name == n {
				found = true
			}
		}

		assert.True(t, found)
	}
}
