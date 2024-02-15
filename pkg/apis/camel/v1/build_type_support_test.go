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

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMatchingBuildsPending(t *testing.T) {
	buildA := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildA",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:timer",
							"camel:log",
						},
					},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhaseScheduling,
		},
	}
	buildB := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildB",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:timer",
							"camel:log",
							"camel:bean",
						},
					},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhasePending,
		},
	}
	buildC := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildC",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:timer",
							"camel:log",
							"camel:bean",
							"camel:zipfile",
						},
					},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhasePending,
		},
	}
	buildZ := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildZ",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:mongodb",
							"camel:component-a",
							"camel:component-b",
						},
					},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhasePending,
		},
	}

	buildList := BuildList{
		Items: []Build{buildA, buildB, buildC, buildZ},
	}

	// buildA is completed, no need to check it
	matches, buildMatch := buildList.HasMatchingBuild(&buildB)
	assert.Equal(t, true, matches)
	assert.Equal(t, buildA.Name, buildMatch.Name)
	matches, buildMatch = buildList.HasMatchingBuild(&buildC)
	assert.Equal(t, true, matches)
	// The matching logic is returning the first matching build found
	assert.True(t, buildMatch.Name == buildA.Name || buildMatch.Name == buildB.Name)
	matches, buildMatch = buildList.HasMatchingBuild(&buildZ)
	assert.Equal(t, false, matches)
	assert.Nil(t, buildMatch)
}
