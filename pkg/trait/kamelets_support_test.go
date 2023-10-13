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
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
)

func TestKameletBundleSingle(t *testing.T) {
	kb := newKameletBundle()
	kb.add(kamelet("my-ns", "test"))
	cmBundle, err := kb.toConfigmaps("my-it", "my-ns")
	assert.Nil(t, err)
	assert.NotNil(t, cmBundle)
	assert.Len(t, cmBundle, 1)
	assert.Equal(t, "my-ns", cmBundle[0].Namespace)
	assert.Equal(t, "kamelets-bundle-my-it-001", cmBundle[0].Name)
	assert.Equal(t, "my-it", cmBundle[0].Labels[v1.IntegrationLabel])
	assert.NotNil(t, cmBundle[0].Data["test.kamelet.yaml"])
}

func TestKameletBundleMultiKameletsSingleConfigmap(t *testing.T) {
	kb := newKameletBundle()
	kb.add(kamelet("default", "test1"))
	kb.add(kamelet("default", "test2"))
	kb.add(kamelet("default", "test3"))
	kb.add(kamelet("default", "test4"))
	kb.add(kamelet("default", "test5"))
	kb.add(kamelet("default", "test6"))
	cmBundle, err := kb.toConfigmaps("my-it", "default")
	assert.Nil(t, err)
	assert.NotNil(t, cmBundle)
	assert.Len(t, cmBundle, 1)
	assert.Equal(t, "default", cmBundle[0].Namespace)
	assert.Equal(t, "kamelets-bundle-my-it-001", cmBundle[0].Name)
	assert.Equal(t, "my-it", cmBundle[0].Labels[v1.IntegrationLabel])
	assert.Len(t, cmBundle[0].Data, 6)
	assert.NotNil(t, cmBundle[0].Data["test1.kamelet.yaml"])
	assert.NotNil(t, cmBundle[0].Data["test2.kamelet.yaml"])
	assert.NotNil(t, cmBundle[0].Data["test3.kamelet.yaml"])
	assert.NotNil(t, cmBundle[0].Data["test4.kamelet.yaml"])
	assert.NotNil(t, cmBundle[0].Data["test5.kamelet.yaml"])
	assert.NotNil(t, cmBundle[0].Data["test6.kamelet.yaml"])
}

func TestKameletBundleMultiKameletsMultiConfigmap(t *testing.T) {
	kb := newKameletBundle()
	for i := 0; i < 2000; i++ {
		kb.add(kamelet("default", fmt.Sprintf("test%d", i)))
	}
	cmBundle, err := kb.toConfigmaps("my-it", "default")
	assert.Nil(t, err)
	assert.NotNil(t, cmBundle)
	assert.Len(t, cmBundle, 2)
	assert.Equal(t, "default", cmBundle[0].Namespace)
	assert.Equal(t, "kamelets-bundle-my-it-001", cmBundle[0].Name)
	assert.Equal(t, "my-it", cmBundle[0].Labels[v1.IntegrationLabel])
	assert.Equal(t, "default", cmBundle[1].Namespace)
	assert.Equal(t, "kamelets-bundle-my-it-002", cmBundle[1].Name)
	assert.Equal(t, "my-it", cmBundle[1].Labels[v1.IntegrationLabel])
	assert.Equal(t, 2000, len(cmBundle[0].Data)+len(cmBundle[1].Data))
	assert.NotNil(t, cmBundle[0].Data["test1.kamelet.yaml"])
	assert.NotNil(t, cmBundle[1].Data["test1999.kamelet.yaml"])
}

func kamelet(ns, name string) *v1.Kamelet {
	kamelet := v1.NewKamelet(ns, name)
	kamelet.Spec = v1.KameletSpec{
		Sources: []v1.SourceSpec{
			{
				DataSpec: v1.DataSpec{
					Name: "mykamelet.groovy",
					Content: `from("timer1").to("log:info")
					from("timer2").to("log:info")
					from("timer3").to("log:info")
					from("timer4").to("log:info")
					from("timer5").to("log:info")
					from("timer6").to("log:info")
					from("timer7").to("log:info")
					from("timer8").to("log:info")
					from("timer9").to("log:info")
					from("timer10").to("log:info")
					from("timer11").to("log:info")
					from("timer12").to("log:info")
					from("timer13").to("log:info")
					from("timer14").to("log:info")
					from("timer15").to("log:info")
					from("timer16").to("log:info")
					from("timer17").to("log:info")`,
				},
				Type: v1.SourceTypeTemplate,
			},
		},
	}
	kamelet.Status = v1.KameletStatus{Phase: v1.KameletPhaseReady}

	return &kamelet
}
