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

package builder

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

type kitOptions struct {
	operatorID   string
	dependencies []string
	traits       []string
}

func TestKitMaxBuildLimit(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		createOperator(ns, "500Mi", "8m0s", "--global", "--force")

		pl := Platform(ns)()
		// set maximum number of running builds
		pl.Spec.Build.MaxRunningBuilds = 2
		if err := TestClient().Update(TestContext, pl); err != nil {
			t.Error(err)
			t.FailNow()
		}

		buildA := "integration-a"
		buildB := "integration-b"
		buildC := "integration-c"

		doKitBuildInNamespace(buildA, ns, TestTimeoutShort, kitOptions{
			operatorID: fmt.Sprintf("camel-k-%s", ns),
			dependencies: []string{
				"camel:timer", "camel:log",
			},
			traits: []string{
				"builder.properties=build-property=A",
			},
		}, v1.BuildPhaseRunning, v1.IntegrationKitPhaseBuildRunning)

		ns1 := NewTestNamespace(false).GetName()
		defer DumpNamespace(t, ns1)
		defer DeleteNamespace(t, ns1)

		pl1 := v1.NewIntegrationPlatform(ns1, fmt.Sprintf("camel-k-%s", ns))
		pl.Spec.DeepCopyInto(&pl1.Spec)
		pl1.Spec.Build.Maven.Settings = v1.ValueSource{}
		pl1.SetOperatorID(fmt.Sprintf("camel-k-%s", ns))
		if err := TestClient().Create(TestContext, &pl1); err != nil {
			t.Error(err)
			t.FailNow()
		}

		doKitBuildInNamespace(buildB, ns1, TestTimeoutShort, kitOptions{
			operatorID: fmt.Sprintf("camel-k-%s", ns),
			dependencies: []string{
				"camel:timer", "camel:log",
			},
			traits: []string{
				"builder.properties=build-property=B",
			},
		}, v1.BuildPhaseRunning, v1.IntegrationKitPhaseBuildRunning)

		ns2 := NewTestNamespace(false).GetName()
		defer DumpNamespace(t, ns2)
		defer DeleteNamespace(t, ns2)

		pl2 := v1.NewIntegrationPlatform(ns2, fmt.Sprintf("camel-k-%s", ns))
		pl.Spec.DeepCopyInto(&pl2.Spec)
		pl2.Spec.Build.Maven.Settings = v1.ValueSource{}
		pl2.SetOperatorID(fmt.Sprintf("camel-k-%s", ns))
		if err := TestClient().Create(TestContext, &pl2); err != nil {
			t.Error(err)
			t.FailNow()
		}

		doKitBuildInNamespace(buildC, ns2, TestTimeoutShort, kitOptions{
			operatorID: fmt.Sprintf("camel-k-%s", ns),
			dependencies: []string{
				"camel:timer", "camel:log",
			},
			traits: []string{
				"builder.properties=build-property=C",
			},
		}, v1.BuildPhaseScheduling, v1.IntegrationKitPhaseNone)

		// verify that buildC is allowed to build as soon as buildA has finished
		Eventually(BuildPhase(ns, buildA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(BuildPhase(ns2, buildC), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))
		Eventually(KitPhase(ns2, buildC), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseBuildRunning))

		// verify that all builds are successful
		Eventually(BuildPhase(ns1, buildB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(KitPhase(ns1, buildB), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseReady))
		Eventually(BuildPhase(ns2, buildC), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(KitPhase(ns2, buildC), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseReady))
	})
}

func TestKitTimerToLogFullBuild(t *testing.T) {
	doKitFullBuild(t, "timer-to-log", "500Mi", "8m0s", TestTimeoutLong, kitOptions{
		dependencies: []string{
			"camel:timer", "camel:log",
		},
	}, v1.BuildPhaseSucceeded, v1.IntegrationKitPhaseReady)
}

func TestKitKnativeFullBuild(t *testing.T) {
	doKitFullBuild(t, "knative", "500Mi", "8m0s", TestTimeoutLong, kitOptions{
		dependencies: []string{
			"camel-quarkus-knative",
		},
	}, v1.BuildPhaseSucceeded, v1.IntegrationKitPhaseReady)
}

func TestKitTimerToLogFullNativeBuild(t *testing.T) {
	doKitFullBuild(t, "timer-to-log", "4Gi", "15m0s", TestTimeoutLong*3, kitOptions{
		dependencies: []string{
			"camel:timer", "camel:log",
		},
		traits: []string{
			"quarkus.package-type=native",
		},
	}, v1.BuildPhaseSucceeded, v1.IntegrationKitPhaseReady)
}

func doKitFullBuild(t *testing.T, name string, memoryLimit string, buildTimeout string, testTimeout time.Duration,
	options kitOptions, buildPhase v1.BuildPhase, kitPhase v1.IntegrationKitPhase) {
	t.Helper()

	WithNewTestNamespace(t, func(ns string) {
		createOperator(ns, memoryLimit, buildTimeout)
		doKitBuildInNamespace(name, ns, testTimeout, options, buildPhase, kitPhase)
	})
}

func createOperator(ns string, memoryLimit string, buildTimeout string, installArgs ...string) {
	strategy := os.Getenv("KAMEL_INSTALL_BUILD_PUBLISH_STRATEGY")
	ocp, err := openshift.IsOpenShift(TestClient())
	Expect(err).To(Succeed())

	args := []string{"--build-timeout", buildTimeout}
	// TODO: configure build Pod resources if applicable
	if strategy == "Spectrum" || ocp {
		args = append(args, "--operator-resources", "limits.memory="+memoryLimit)
	}

	args = append(args, installArgs...)

	operatorID := fmt.Sprintf("camel-k-%s", ns)
	Expect(KamelInstallWithID(operatorID, ns, args...).Execute()).To(Succeed())
}

func doKitBuildInNamespace(name string, ns string, testTimeout time.Duration, options kitOptions, buildPhase v1.BuildPhase, kitPhase v1.IntegrationKitPhase) {

	buildKitArgs := []string{"kit", "create", name, "-n", ns}
	for _, dependency := range options.dependencies {
		buildKitArgs = append(buildKitArgs, "-d", dependency)
	}
	for _, trait := range options.traits {
		buildKitArgs = append(buildKitArgs, "-t", trait)
	}

	if options.operatorID != "" {
		buildKitArgs = append(buildKitArgs, "--operator-id", options.operatorID)
	} else {
		buildKitArgs = append(buildKitArgs, "--operator-id", fmt.Sprintf("camel-k-%s", ns))
	}

	Expect(Kamel(buildKitArgs...).Execute()).To(Succeed())

	Eventually(Build(ns, name), testTimeout).ShouldNot(BeNil())
	if buildPhase != v1.BuildPhaseNone {
		Eventually(BuildPhase(ns, name), testTimeout).Should(Equal(buildPhase))
	}
	if kitPhase != v1.IntegrationKitPhaseNone {
		Eventually(KitPhase(ns, name), testTimeout).Should(Equal(kitPhase))
	}
}
