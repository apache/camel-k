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
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

type kitOptions struct {
	dependencies []string
	traits       []string
}

func TestKitTimerToLogFullBuild(t *testing.T) {
	doKitFullBuild(t, "timer-to-log", "300Mi", "5m0s", TestTimeoutLong, kitOptions{
		dependencies: []string{
			"camel:timer", "camel:log",
		},
	})
}

func TestKitKnativeFullBuild(t *testing.T) {
	doKitFullBuild(t, "knative", "300Mi", "5m0s", TestTimeoutLong, kitOptions{
		dependencies: []string{
			"camel-k-knative",
		},
	})
}

func TestKitTimerToLogFullNativeBuild(t *testing.T) {
	doKitFullBuild(t, "timer-to-log", "4Gi", "15m0s", TestTimeoutVeryLong, kitOptions{
		dependencies: []string{
			"camel:timer", "camel:log",
		},
		traits: []string{
			"quarkus.package-type=native",
		},
	})
}

func doKitFullBuild(t *testing.T, name string, memoryLimit string, buildTimeout string, testTimeout time.Duration, options kitOptions) {
	t.Helper()

	WithNewTestNamespace(t, func(ns string) {
		strategy := os.Getenv("KAMEL_INSTALL_BUILD_PUBLISH_STRATEGY")
		ocp, err := openshift.IsOpenShift(TestClient())
		Expect(err).To(Succeed())

		args := []string{"--build-timeout", buildTimeout}
		// TODO: configure build Pod resources if applicable
		if strategy == "Spectrum" || ocp {
			args = append(args, "--operator-resources", "limits.memory="+memoryLimit)
		}

		Expect(KamelInstall(ns, args...).Execute()).To(Succeed())

		buildKitArgs := []string{"kit", "create", name, "-n", ns}
		for _, dependency := range options.dependencies {
			buildKitArgs = append(buildKitArgs, "-d", dependency)
		}
		for _, trait := range options.traits {
			buildKitArgs = append(buildKitArgs, "-t", trait)
		}
		Expect(Kamel(buildKitArgs...).Execute()).To(Succeed())

		Eventually(Build(ns, name), testTimeout).ShouldNot(BeNil())
		Eventually(BuildPhase(ns, name), testTimeout).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(KitPhase(ns, name), testTimeout).Should(Equal(v1.IntegrationKitPhaseReady))
	})
}
