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
	"fmt"
	"reflect"
)

var stepsByID = make(map[string]Step)

type builderStep struct {
	StepID string
	phase  int32
	task   StepTask
}

func (s *builderStep) String() string {
	return fmt.Sprintf("%s@%d", s.StepID, s.phase)
}

func (s *builderStep) ID() string {
	return s.StepID
}

func (s *builderStep) Phase() int32 {
	return s.phase
}

func (s *builderStep) execute(ctx *builderContext) error {
	return s.task(ctx)
}

type StepTask func(*builderContext) error

func NewStep(phase int32, task StepTask) Step {
	s := builderStep{
		phase: phase,
		task:  task,
	}

	return &s
}

func StepsFrom(ids ...string) ([]Step, error) {
	steps := make([]Step, 0)
	for _, id := range ids {
		s, ok := stepsByID[id]
		if !ok {
			return steps, fmt.Errorf("unknown build step: %s", id)
		}
		steps = append(steps, s)
	}
	return steps, nil
}

func StepIDsFor(steps ...Step) []string {
	IDs := make([]string, 0)
	for _, step := range steps {
		IDs = append(IDs, step.ID())
	}
	return IDs
}

func registerSteps(steps interface{}) {
	v := reflect.ValueOf(steps)
	t := reflect.TypeOf(steps)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if step, ok := v.Field(i).Interface().(Step); ok {
			id := t.PkgPath() + "/" + field.Name
			// Set the fully qualified step ID
			reflect.Indirect(v.Field(i).Elem()).FieldByName("StepID").SetString(id)

			registerStep(step)
		}
	}
}

func registerStep(steps ...Step) {
	for _, step := range steps {
		if _, exists := stepsByID[step.ID()]; exists {
			panic(fmt.Errorf("the build step is already registered: %s", step.ID()))
		}
		stepsByID[step.ID()] = step
	}
}
