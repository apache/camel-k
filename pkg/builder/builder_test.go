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

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/test"
)

type errorTestSteps struct {
	Step1 Step
	Step2 Step
}

func TestFailure(t *testing.T) {
	c, err := test.NewFakeClient()
	assert.Nil(t, err)

	b := New(c)

	steps := errorTestSteps{
		Step1: NewStep(InitPhase, func(i *builderContext) error {
			return nil
		}),
		Step2: NewStep(ApplicationPublishPhase, func(i *builderContext) error {
			return errors.New("an error")
		}),
	}

	registerSteps(steps)

	build := &v1.Build{
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name: "builder",
						},
						Steps: StepIDsFor(
							steps.Step1,
							steps.Step2,
						),
					},
				},
			},
		},
	}

	ctx := cancellable.NewContext()
	status := b.Build(build).TaskByName("builder").Do(ctx)
	assert.Equal(t, v1.BuildPhaseFailed, status.Phase)
	assert.Equal(t, "an error", status.Error)
}
