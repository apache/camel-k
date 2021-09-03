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
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
)

type kitOptions struct {
	dependencies []string
	traits       []string
}

func TestKitTimerToLogFullBuild(t *testing.T) {
	doKitFullBuild(t, "timer-to-log", kitOptions{
		dependencies: []string{
			"camel:timer", "camel:log",
		},
	})
}

func TestKitKnativeFullBuild(t *testing.T) {
	doKitFullBuild(t, "knative", kitOptions{
		dependencies: []string{
			"camel:knative",
		},
	})
}

func TestKitTimerToLogFullNativeBuild(t *testing.T) {
	doKitFullBuild(t, "timer-to-log", kitOptions{
		dependencies: []string{
			"camel:timer", "camel:log",
		},
		traits: []string{
			"quarkus.package-type=native",
		},
	})
}

func doKitFullBuild(t *testing.T, name string, options kitOptions) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		buildKitArgs := []string{"kit", "create", name, "-n", ns}
		for _, dependency := range options.dependencies {
			buildKitArgs = append(buildKitArgs, "-d", dependency)
		}
		for _, trait := range options.traits {
			buildKitArgs = append(buildKitArgs, "-t", trait)
		}
		Expect(Kamel(buildKitArgs...).Execute()).To(Succeed())
		Eventually(Build(ns, name)).ShouldNot(BeNil())
		Eventually(BuildPhase(ns, name), TestTimeoutMedium).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(KitPhase(ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationKitPhaseReady))
	})
}
