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

package build

import (
	"context"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewBuildPodConfiguration(t *testing.T) {

	ctx := context.TODO()
	c, err := test.NewFakeClient()
	assert.Nil(t, err)

	build := v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: "theBuildName",
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name: "builder",
							Configuration: v1.BuildConfiguration{
								BuilderPodNamespace: "theNamespace",
								NodeSelector:        map[string]string{"node": "selector"},
								Annotations:         map[string]string{"annotation": "value"},
							},
						},
					},
				},
			},
		},
	}

	pod := newBuildPod(ctx, c, &build)

	assert.Equal(t, "Pod", pod.Kind)
	assert.Equal(t, "theNamespace", pod.Namespace)
	assert.Equal(t, map[string]string{
		"camel.apache.org/build":     "theBuildName",
		"camel.apache.org/component": "builder",
	}, pod.Labels)
	assert.Equal(t, map[string]string{"node": "selector"}, pod.Spec.NodeSelector)
	assert.Equal(t, map[string]string{"annotation": "value"}, pod.Annotations)
}
