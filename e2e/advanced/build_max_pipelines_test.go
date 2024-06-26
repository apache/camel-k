//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package advanced

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

type kitOptions struct {
	dependencies []string
	traits       []string
}

func kitMaxBuildLimit(t *testing.T, maxRunningBuilds int32, buildOrderStrategy v1.BuildOrderStrategy) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, g, ns)

		pl := Platform(t, ctx, ns)()
		// set maximum number of running builds and order strategy
		pl.Spec.Build.MaxRunningBuilds = maxRunningBuilds
		pl.Spec.Build.BuildConfiguration.OrderStrategy = buildOrderStrategy
		if err := TestClient(t).Update(ctx, pl); err != nil {
			t.Error(err)
			t.FailNow()
		}

		buildA := "integration-a"
		buildB := "integration-b"
		buildC := "integration-c"

		doKitBuildInNamespace(t, ctx, g, buildA, ns, TestTimeoutShort, kitOptions{
			dependencies: []string{
				"camel:timer", "camel:log",
			},
			traits: []string{
				"builder.properties=build-property=A",
			},
		}, v1.BuildPhaseRunning, v1.IntegrationKitPhaseBuildRunning)

		doKitBuildInNamespace(t, ctx, g, buildB, ns, TestTimeoutShort, kitOptions{
			dependencies: []string{
				"camel:timer", "camel:log",
			},
			traits: []string{
				"builder.properties=build-property=B",
			},
		}, v1.BuildPhaseRunning, v1.IntegrationKitPhaseBuildRunning)

		doKitBuildInNamespace(t, ctx, g, buildC, ns, TestTimeoutShort, kitOptions{
			dependencies: []string{
				"camel:timer", "camel:log",
			},
			traits: []string{
				"builder.properties=build-property=C",
			},
		}, v1.BuildPhaseScheduling, v1.IntegrationKitPhaseNone)

		var notExceedsMaxBuildLimit = func(runningBuilds int) bool {
			return runningBuilds <= 2
		}

		limit := 0
		for limit < 5 && BuildPhase(t, ctx, ns, buildA)() == v1.BuildPhaseRunning {
			// verify that number of running builds does not exceed max build limit
			g.Consistently(BuildsRunning(BuildPhase(t, ctx, ns, buildA), BuildPhase(t, ctx, ns, buildB), BuildPhase(t, ctx, ns, buildC)), TestTimeoutShort, 10*time.Second).Should(Satisfy(notExceedsMaxBuildLimit))
			limit++
		}

		// make sure we have verified max build limit at least once
		if limit == 0 {
			t.Error(errors.New(fmt.Sprintf("Unexpected build phase '%s' for %s - not able to verify max builds limit", BuildPhase(t, ctx, ns, buildA)(), buildA)))
			t.FailNow()
		}

		// verify that all builds are successful
		g.Eventually(BuildPhase(t, ctx, ns, buildA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(KitPhase(t, ctx, ns, buildA), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseReady))
		g.Eventually(BuildPhase(t, ctx, ns, buildB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(KitPhase(t, ctx, ns, buildB), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseReady))
		g.Eventually(BuildPhase(t, ctx, ns, buildC), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(KitPhase(t, ctx, ns, buildC), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseReady))
	})
}

func TestKitMaxBuildLimitSequential(t *testing.T) {
	kitMaxBuildLimit(t, 2, v1.BuildOrderStrategySequential)
}

func TestKitMaxBuildLimitFIFO(t *testing.T) {
	kitMaxBuildLimit(t, 2, v1.BuildOrderStrategyFIFO)
}

func TestKitMaxBuildLimitDependencies(t *testing.T) {
	kitMaxBuildLimit(t, 2, v1.BuildOrderStrategyDependencies)
}

func TestMaxBuildLimitWaitingBuilds(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, g, ns)

		pl := Platform(t, ctx, ns)()
		// set maximum number of running builds and order strategy
		pl.Spec.Build.MaxRunningBuilds = 1
		pl.Spec.Build.BuildConfiguration.OrderStrategy = v1.BuildOrderStrategyFIFO
		if err := TestClient(t).Update(ctx, pl); err != nil {
			t.Error(err)
			t.FailNow()
		}

		buildA := "integration-a"
		buildB := "integration-b"
		buildC := "integration-c"

		doKitBuildInNamespace(t, ctx, g, buildA, ns, TestTimeoutShort, kitOptions{
			dependencies: []string{
				"camel:timer", "camel:log",
			},
			traits: []string{
				"builder.properties=build-property=A",
			},
		}, v1.BuildPhaseRunning, v1.IntegrationKitPhaseBuildRunning)

		doKitBuildInNamespace(t, ctx, g, buildB, ns, TestTimeoutShort, kitOptions{
			dependencies: []string{
				"camel:cron", "camel:log", "camel:joor",
			},
			traits: []string{
				"builder.properties=build-property=B",
			},
		}, v1.BuildPhaseScheduling, v1.IntegrationKitPhaseNone)

		doKitBuildInNamespace(t, ctx, g, buildC, ns, TestTimeoutShort, kitOptions{
			dependencies: []string{
				"camel:timer", "camel:log", "camel:joor", "camel:http",
			},
			traits: []string{
				"builder.properties=build-property=C",
			},
		}, v1.BuildPhaseScheduling, v1.IntegrationKitPhaseNone)

		// verify that last build is waiting
		g.Eventually(BuildConditions(t, ctx, ns, buildC), TestTimeoutMedium).ShouldNot(BeNil())
		g.Eventually(
			BuildCondition(t, ctx, ns, buildC, v1.BuildConditionType(v1.BuildConditionScheduled))().Status,
			TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		g.Eventually(
			BuildCondition(t, ctx, ns, buildC, v1.BuildConditionType(v1.BuildConditionScheduled))().Reason,
			TestTimeoutShort).Should(Equal(v1.BuildConditionWaitingReason))

		// verify that last build is scheduled
		g.Eventually(BuildPhase(t, ctx, ns, buildC), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(KitPhase(t, ctx, ns, buildC), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseReady))

		g.Eventually(BuildConditions(t, ctx, ns, buildC), TestTimeoutLong).ShouldNot(BeNil())
		g.Eventually(
			BuildCondition(t, ctx, ns, buildC, v1.BuildConditionType(v1.BuildConditionScheduled))().Status,
			TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(
			BuildCondition(t, ctx, ns, buildC, v1.BuildConditionType(v1.BuildConditionScheduled))().Reason,
			TestTimeoutShort).Should(Equal(v1.BuildConditionReadyReason))
	})
}

func doKitBuildInNamespace(t *testing.T, ctx context.Context, g *WithT, name string, ns string, testTimeout time.Duration, options kitOptions, buildPhase v1.BuildPhase, kitPhase v1.IntegrationKitPhase) {
	buildKitArgs := []string{"kit", "create", name, "-n", ns}
	for _, dependency := range options.dependencies {
		buildKitArgs = append(buildKitArgs, "-d", dependency)
	}
	for _, trait := range options.traits {
		buildKitArgs = append(buildKitArgs, "-t", trait)
	}

	g.Expect(Kamel(t, ctx, buildKitArgs...).Execute()).To(Succeed())

	g.Eventually(Build(t, ctx, ns, name), testTimeout).ShouldNot(BeNil())
	if buildPhase != v1.BuildPhaseNone {
		g.Eventually(BuildPhase(t, ctx, ns, name), testTimeout).Should(Equal(buildPhase))
	}
	if kitPhase != v1.IntegrationKitPhaseNone {
		g.Eventually(KitPhase(t, ctx, ns, name), testTimeout).Should(Equal(kitPhase))
	}
}
