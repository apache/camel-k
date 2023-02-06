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

package label

import (
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
)

func TestParseValidEntries(t *testing.T) {
	integration := "test1"
	AdditionalLabels = "k1=v1,k3/k3.k3=v3.v3,k4.k4=v4,k5/k5=v5"
	FixedLabels = map[string]string{}
	checkAdditionalLabels()
	labels := AddLabels(integration)
	expected := map[string]string{
		v1.IntegrationLabel: integration,
		"k1":                "v1",
		"k3/k3.k3":          "v3.v3",
		"k4.k4":             "v4",
		"k5/k5":             "v5",
	}
	assert.Equal(t, expected, labels)

	AdditionalLabels = "k1=v1"
	FixedLabels = map[string]string{}
	checkAdditionalLabels()
	labels = AddLabels(integration)
	expected = map[string]string{
		v1.IntegrationLabel: integration,
		"k1":                "v1",
	}
	assert.Equal(t, expected, labels)
}

func TestParseEmptyAdditionalLabels(t *testing.T) {
	integration := "test1"
	AdditionalLabels = ""
	FixedLabels = map[string]string{}
	checkAdditionalLabels()
	labels := AddLabels(integration)
	expected := map[string]string{
		v1.IntegrationLabel: integration,
	}
	assert.Equal(t, expected, labels)
}

func TestParseInvalidEntry(t *testing.T) {
	integration := "test1"
	AdditionalLabels = "k1[=v1,k2)=v2,k@3=v3"
	FixedLabels = map[string]string{}
	assert.Panics(t, func() {
		checkAdditionalLabels()
		AddLabels(integration)
	})
}

func TestParseIntegrationPlaceholder(t *testing.T) {
	integration := "test1"
	AdditionalLabels = "k1=token_integration_name,k2=v2,k3=v3,k4.k4=v4,k5/k5=v5,rht.subcomp_t=my_subcomp"
	FixedLabels = map[string]string{}
	checkAdditionalLabels()
	labels := AddLabels(integration)
	expected := map[string]string{
		v1.IntegrationLabel: integration,
		"k1":                integration,
		"k2":                "v2",
		"k3":                "v3",
		"k4.k4":             "v4",
		"k5/k5":             "v5",
		"rht.subcomp_t":     "my_subcomp",
	}
	assert.Equal(t, expected, labels)

	AdditionalLabels = "k1=v1,k2=v2,k3=token_integration_name"
	FixedLabels = map[string]string{}
	checkAdditionalLabels()
	labels = AddLabels(integration)
	expected = map[string]string{
		v1.IntegrationLabel: integration,
		"k1":                "v1",
		"k2":                "v2",
		"k3":                integration,
	}
	assert.Equal(t, expected, labels)

	AdditionalLabels = "k3=token_integration_name"
	FixedLabels = map[string]string{}
	checkAdditionalLabels()
	labels = AddLabels(integration)
	expected = map[string]string{
		v1.IntegrationLabel: integration,
		"k3":                integration,
	}
	assert.Equal(t, expected, labels)
}
