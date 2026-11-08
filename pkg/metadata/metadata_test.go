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

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	src "github.com/apache/camel-k/v2/pkg/util/source"
)

func TestMergeBeanDefinitions(t *testing.T) {
	first := src.NewMetadata()
	first.Beans = []src.BeanDefinition{
		{
			Name:       "inventoryDataSource",
			Type:       "org.example.InventoryDataSource",
			Properties: map[string]string{"host": "db.internal"},
		},
	}
	second := src.NewMetadata()
	second.Beans = []src.BeanDefinition{
		{
			Name:       "metricsRegistry",
			Type:       "com.example.MetricsRegistry",
			Properties: map[string]string{"host": "another-db.internal"},
		},
	}

	merged := merge(first, second)

	assert.Equal(t, []src.BeanDefinition{
		first.Beans[0],
		second.Beans[0],
	}, merged.Beans)
}
