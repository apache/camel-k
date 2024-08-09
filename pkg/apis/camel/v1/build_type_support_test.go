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
	"time"

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
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
	assert.True(t, matches)
	assert.Equal(t, buildA.Name, buildMatch.Name)
	matches, buildMatch = buildList.HasMatchingBuild(&buildC)
	assert.True(t, matches)
	// The matching logic is returning the first matching build found
	assert.True(t, buildMatch.Name == buildA.Name || buildMatch.Name == buildB.Name)
	matches, buildMatch = buildList.HasMatchingBuild(&buildZ)
	assert.False(t, matches)
	assert.Nil(t, buildMatch)
}

func TestMatchingBuildsSchedulingSharedDependencies(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02T15:04:05-0700", "2024-08-09T10:00:00Z")
	creationTimestamp := v1.Time{Time: timestamp}
	buildA := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildA",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:core",
							"camel:rest",
							"mvn:org.apache.camel.k:camel-k-runtime",
							"mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl",
						},
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
			Name:              "buildB",
			CreationTimestamp: creationTimestamp,
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:quartz",
							"mvn:org.apache.camel.k:camel-k-cron",
							"mvn:org.apache.camel.k:camel-k-runtime",
							"mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl",
						},
						Runtime: RuntimeSpec{
							Version: "3.8.1",
						}},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhaseScheduling,
		},
	}

	buildList := BuildList{
		Items: []Build{buildA, buildB},
	}

	// both builds share dependencies and have the same creationTimestamp
	// buildA should be prioritized so there should be not matching build for it

	matches, buildMatch := buildList.HasMatchingBuild(&buildA)
	assert.False(t, matches)
	assert.Nil(t, buildMatch)
	matches, buildMatch = buildList.HasMatchingBuild(&buildB)
	assert.True(t, matches)
	assert.True(t, buildMatch.Name == buildA.Name)
}

func TestMatchingBuildsSchedulingSameDependenciesDIfferentRuntimes(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02T15:04:05-0700", "2024-08-09T10:00:00Z")
	creationTimestamp := v1.Time{Time: timestamp}
	buildA := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildA",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:quartz",
							"mvn:org.apache.camel.k:camel-k-cron",
							"mvn:org.apache.camel.k:camel-k-runtime",
							"mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl",
						},
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
			Name:              "buildB",
			CreationTimestamp: creationTimestamp,
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:quartz",
							"mvn:org.apache.camel.k:camel-k-cron",
							"mvn:org.apache.camel.k:camel-k-runtime",
							"mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl",
						},
						Runtime: RuntimeSpec{
							Version: "3.2.3",
						},
					},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhaseScheduling,
		},
	}

	buildList := BuildList{
		Items: []Build{buildA, buildB},
	}

	// each build uses a different runtime, so they should not match

	matches, buildMatch := buildList.HasMatchingBuild(&buildA)
	assert.False(t, matches)
	assert.Nil(t, buildMatch)
	matches, buildMatch = buildList.HasMatchingBuild(&buildB)
	assert.False(t, matches)
	assert.Nil(t, buildMatch)
}

func TestMatchingBuildsSchedulingSameDependenciesSameRuntime(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02T15:04:05-0700", "2024-08-09T10:00:00Z")
	creationTimestamp := v1.Time{Time: timestamp}
	buildA := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildA",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:quartz",
							"mvn:org.apache.camel.k:camel-k-cron",
							"mvn:org.apache.camel.k:camel-k-runtime",
							"mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl",
						},
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
			Name:              "buildB",
			CreationTimestamp: creationTimestamp,
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:quartz",
							"mvn:org.apache.camel.k:camel-k-cron",
							"mvn:org.apache.camel.k:camel-k-runtime",
							"mvn:org.apache.camel.quarkus:camel-quarkus-yaml-dsl",
						},
						Runtime: RuntimeSpec{
							Version: "3.8.1",
						},
					},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhaseScheduling,
		},
	}

	buildList := BuildList{
		Items: []Build{buildA, buildB},
	}

	// ebuilds have the same dependencies, runtime and creation timestamp

	matches, buildMatch := buildList.HasMatchingBuild(&buildA)
	assert.False(t, matches)
	assert.Nil(t, buildMatch)
	matches, buildMatch = buildList.HasMatchingBuild(&buildB)
	assert.True(t, matches)
	assert.True(t, buildMatch.Name == buildA.Name)
}

func TestMatchingBuildsSchedulingFewCommonDependencies(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02T15:04:05-0700", "2024-08-09T10:00:00Z")
	creationTimestamp := v1.Time{Time: timestamp}
	buildA := Build{
		ObjectMeta: v1.ObjectMeta{
			Name: "buildA",
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:quartz",
							"camel:componenta1",
							"camel:componentb1",
							"camel:componentc1",
							"camel:componentd1",
							"camel:componente1",
							"camel:componentf1",
							"camel:componentg1",
							"camel:componenth1",
							"camel:componenti1",
						},
						Runtime: RuntimeSpec{
							Version: "3.8.1",
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
			Name:              "buildB",
			CreationTimestamp: creationTimestamp,
		},
		Spec: BuildSpec{
			Tasks: []Task{
				{
					Builder: &BuilderTask{
						Dependencies: []string{
							"camel:quartz",
							"camel:componenta2",
							"camel:componentb2",
							"camel:componentc2",
							"camel:componentd2",
							"camel:componente2",
							"camel:componentf2",
							"camel:componentg2",
							"camel:componenth2",
							"camel:componenti2",
						},
						Runtime: RuntimeSpec{
							Version: "3.8.1",
						},
					},
				},
			},
		},
		Status: BuildStatus{
			Phase: BuildPhaseScheduling,
		},
	}

	buildList := BuildList{
		Items: []Build{buildA, buildB},
	}

	// builds have only 1 out of 10 shared dependencies. they should not match

	matches, buildMatch := buildList.HasMatchingBuild(&buildA)
	assert.False(t, matches)
	assert.Nil(t, buildMatch)
	matches, buildMatch = buildList.HasMatchingBuild(&buildB)
	assert.False(t, matches)
	assert.Nil(t, buildMatch)
}
