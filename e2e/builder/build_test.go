// +build integration

// To enable compilation of this file in Goland, go to "File -> Settings -> Go -> Build Tags & Vendoring -> Build Tags -> Custom tags" and add "integration"

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

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
)

func TestKitTimerToLogFullBuild(t *testing.T) {
	doKitFullBuild(t, "timer-to-log", "camel:timer", "camel:log")
}

func TestKitKnativeFullBuild(t *testing.T) {
	doKitFullBuild(t, "knative", "camel:knative")
}

func doKitFullBuild(t *testing.T, name string, dependencies ...string) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		buildKitArgs := []string{"kit", "create", name, "-n", ns}
		for _, dep := range dependencies {
			buildKitArgs = append(buildKitArgs, "-d", dep)
		}
		Expect(Kamel(buildKitArgs...).Execute()).Should(BeNil())
		Eventually(Build(ns, name)).ShouldNot(BeNil())
		Eventually(func() v1.BuildPhase {
			return Build(ns, name)().Status.Phase
		}, TestTimeoutMedium).Should(Equal(v1.BuildPhaseSucceeded))
	})
}
